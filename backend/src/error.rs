use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Failed to execute nix command: {0}")]
    NixCommandFailed(String),
    #[error("Failed to parse nix output: {0}")]
    NixOutputParseFailed(String),
}

