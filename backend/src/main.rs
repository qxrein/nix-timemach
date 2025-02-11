use chrono::{DateTime, NaiveDateTime, Utc};
use clap::{Command, Subcommand};
use serde::{Serialize, Serializer};
use std::process::Command as StdCommand;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Failed to execute nix command: {0}")]
    NixCommandFailed(String),
    #[error("Failed to parse nix output: {0}")]
    NixOutputParseFailed(String),
    #[error("Failed to parse generation diff: {0}")]
    DiffParseFailed(String),
}

#[derive(Serialize)]
struct Generation {
    id: String,
    #[serde(serialize_with = "serialize_timestamp_as_string")]
    timestamp: DateTime<Utc>,
    description: String,
    profiles: Vec<String>,
}

#[derive(Serialize)]
struct GenerationDiff {
    added: Vec<String>,
    removed: Vec<String>,
    modified: Vec<String>,
}

#[derive(Subcommand)]
enum Commands {
    ListGenerations,
    Diff { from: String, to: String },
}

fn parse_timestamp(date: &str, time: &str) -> Result<DateTime<Utc>, Error> {
    let datetime_str = format!("{} {}", date, time);
    NaiveDateTime::parse_from_str(&datetime_str, "%Y-%m-%d %H:%M:%S")
        .map_err(|e| Error::NixOutputParseFailed(e.to_string()))
        .map(|dt| DateTime::from_naive_utc_and_offset(dt, Utc))
}

fn serialize_timestamp_as_string<S>(
    timestamp: &DateTime<Utc>,
    serializer: S,
) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    serializer.serialize_str(&timestamp.to_rfc3339())
}

fn list_generations() -> Result<Vec<Generation>, Error> {
    let output = StdCommand::new("nixos-rebuild")
        .arg("list-generations")
        .output()
        .map_err(|e| Error::NixCommandFailed(e.to_string()))?;

    if !output.status.success() {
        return Err(Error::NixCommandFailed(
            String::from_utf8_lossy(&output.stderr).to_string(),
        ));
    }

    let output_str = String::from_utf8_lossy(&output.stdout);
    let generations: Vec<Generation> = output_str
        .lines()
        .skip(1) // Skip header
        .filter_map(|line| {
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() >= 4 {
                let id = parts[0].trim_end_matches("current").to_string();
                let date = parts[1];
                let time = parts[2];
                let description = if parts[0].contains("current") {
                    "(current)".to_string()
                } else {
                    "".to_string()
                };

                let timestamp = parse_timestamp(date, time).ok()?;
                let profiles = vec![format!("/nix/var/nix/profiles/system-{}-link", &id)];

                Some(Generation {
                    id,
                    timestamp,
                    description,
                    profiles,
                })
            } else {
                None
            }
        })
        .collect();

    Ok(generations)
}

fn get_diff(from: &str, to: &str) -> Result<GenerationDiff, Error> {
    let from_path = format!("/nix/var/nix/profiles/system-{}-link", from);
    let to_path = format!("/nix/var/nix/profiles/system-{}-link", to);

    let output = StdCommand::new("nix-store")
        .args(["-q", "--references"])
        .arg(&from_path)
        .output()
        .map_err(|e| Error::NixCommandFailed(e.to_string()))?;

    let from_refs: Vec<String> = String::from_utf8_lossy(&output.stdout)
        .lines()
        .map(|s| s.to_string())
        .collect();

    let output = StdCommand::new("nix-store")
        .args(["-q", "--references"])
        .arg(&to_path)
        .output()
        .map_err(|e| Error::NixCommandFailed(e.to_string()))?;

    let to_refs: Vec<String> = String::from_utf8_lossy(&output.stdout)
        .lines()
        .map(|s| s.to_string())
        .collect();

    let added: Vec<String> = to_refs
        .iter()
        .filter(|x| !from_refs.contains(x))
        .cloned()
        .collect();

    let removed: Vec<String> = from_refs
        .iter()
        .filter(|x| !to_refs.contains(x))
        .cloned()
        .collect();

    // For modified, we'll look for packages with the same name but different hashes
    let modified: Vec<String> = from_refs
        .iter()
        .filter(|x| {
            let name = x.split("-").nth(1).unwrap_or("");
            to_refs
                .iter()
                .any(|y| y.split("-").nth(1).unwrap_or("") == name && y != *x)
        })
        .cloned()
        .collect();

    Ok(GenerationDiff {
        added,
        removed,
        modified,
    })
}

fn main() -> Result<(), Error> {
    let cli = Command::new("nix-timemach-backend")
        .version("0.0.1")
        .about("Nix Time Machine")
        .subcommand_required(true)
        .subcommand(Command::new("list-generations").about("List all generations"))
        .subcommand(
            Command::new("diff")
                .about("Show diff between two generations")
                .arg(clap::arg!(<from> "From generation ID"))
                .arg(clap::arg!(<to> "To generation ID")),
        )
        .get_matches();

    match cli.subcommand() {
        Some(("list-generations", _)) => {
            let generations = list_generations()?;
            println!(
                "{}",
                serde_json::to_string(&generations)
                    .map_err(|e| Error::NixOutputParseFailed(e.to_string()))?
            );
        }
        Some(("diff", matches)) => {
            let from = matches.get_one::<String>("from").unwrap();
            let to = matches.get_one::<String>("to").unwrap();
            let diff = get_diff(from, to)?;
            println!(
                "{}",
                serde_json::to_string(&diff)
                    .map_err(|e| Error::NixOutputParseFailed(e.to_string()))?
            );
        }
        _ => unreachable!(),
    }

    Ok(())
}
