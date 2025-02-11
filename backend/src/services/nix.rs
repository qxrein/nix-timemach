use chrono::{DateTime, Utc};
use regex::Regex;
use std::process::Command;

use crate::error::{Error, Result};
use crate::models::diff::GenerationDiff;
use crate::models::generation::Generation;

pub struct NixService;

impl NixService {
    pub fn new() -> Self {
        Self
    }

    pub fn list_generations(&self) -> Result<Vec<Generation>> {
        let output = Command::new("nix-env")
            .args(["--list-generations", "-p", "/nix/var/nix/profiles/system"])
            .output()?;

        if !output.status.success() {
            return Err(Error::NixCommandError(
                String::from_utf8_lossy(&output.stderr).to_string(),
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        self.parse_generations_output(&output_str)
    }

    fn parse_generations_output(&self, output: &str) -> Result<Vec<Generation>> {
        let re = Regex::new(r"^\s*(\d+)\s+(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+(.*)$")
            .map_err(|e| Error::ParseError(e.to_string()))?;

        let current_generation = self.get_current_generation()?;

        let mut generations = Vec::new();
        for line in output.lines() {
            if let Some(caps) = re.captures(line) {
                let id = caps[1].to_string();
                let timestamp = DateTime::parse_from_str(&caps[2], "%Y-%m-%d %H:%M:%S")
                    .map_err(|e| Error::ParseError(e.to_string()))?
                    .with_timezone(&Utc);
                let description = Some(caps[3].trim().to_string());

                generations.push(Generation {
                    id: id.clone(),
                    timestamp,
                    description,
                    profiles: vec![format!("/nix/var/nix/profiles/system-{}-link", id)],
                    current: id == current_generation,
                });
            }
        }

        Ok(generations)
    }

    fn get_current_generation(&self) -> Result<String> {
        let output = Command::new("readlink")
            .args(["/nix/var/nix/profiles/system"])
            .output()?;

        if !output.status.success() {
            return Err(Error::NixCommandError(
                String::from_utf8_lossy(&output.stderr).to_string(),
            ));
        }

        let path = String::from_utf8_lossy(&output.stdout);
        let re = Regex::new(r"system-(\d+)-link").map_err(|e| Error::ParseError(e.to_string()))?;

        if let Some(caps) = re.captures(&path) {
            Ok(caps[1].to_string())
        } else {
            Err(Error::ParseError(
                "Failed to extract current generation ID".into(),
            ))
        }
    }

    pub fn get_diff(&self, from: &str, to: &str) -> Result<GenerationDiff> {
        // Get store paths for both generations
        let from_path = self.get_generation_store_path(from)?;
        let to_path = self.get_generation_store_path(to)?;

        // Use nix-diff to compare the generations
        let output = Command::new("nix-diff")
            .arg(&from_path)
            .arg(&to_path)
            .output()?;

        if !output.status.success() {
            return Err(Error::NixCommandError(
                String::from_utf8_lossy(&output.stderr).to_string(),
            ));
        }

        self.parse_diff_output(&String::from_utf8_lossy(&output.stdout))
    }

    fn get_generation_store_path(&self, id: &str) -> Result<String> {
        let output = Command::new("nix-env")
            .args([
                "-p",
                &format!("/nix/var/nix/profiles/system-{}-link", id),
                "--query",
                "--out-path",
            ])
            .output()?;

        if !output.status.success() {
            return Err(Error::GenerationNotFound(id.to_string()));
        }

        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    }

    fn parse_diff_output(&self, output: &str) -> Result<GenerationDiff> {
        let mut added = Vec::new();
        let mut removed = Vec::new();
        let mut modified = Vec::new();

        for line in output.lines() {
            let line = line.trim();
            if line.starts_with('+') {
                added.push(line[1..].trim().to_string());
            } else if line.starts_with('-') {
                removed.push(line[1..].trim().to_string());
            } else if line.starts_with('~') {
                modified.push(line[1..].trim().to_string());
            }
        }

        Ok(GenerationDiff {
            added,
            removed,
            modified,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_generations_output() {
        let service = NixService::new();
        let sample_output = r#"   1   2024-02-09 10:00:00   nixos-22.11.20240209.123
   2   2024-02-09 11:00:00   nixos-22.11.20240209.456"#;

        let generations = service.parse_generations_output(sample_output).unwrap();
        assert_eq!(generations.len(), 2);
        assert_eq!(generations[0].id, "1");
        assert_eq!(generations[1].id, "2");
    }
}
