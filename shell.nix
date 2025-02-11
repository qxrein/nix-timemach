{ pkgs ? import <nixpkgs> {} }:

let
  # Rust specific dependencies
  rustEnv = with pkgs; [
    cargo
    rustc
    rustfmt
    rust-analyzer
    clippy
    pkg-config
    openssl.dev
  ];

  # Go specific dependencies
  goEnv = with pkgs; [
    go
    gopls
    go-tools
    delve
  ];

  # Development tools
  devTools = with pkgs; [
    git
    gcc
    gnumake
    cmake
    nixfmt
    sqlite
  ];

in
pkgs.mkShell {
  buildInputs = rustEnv ++ goEnv ++ devTools;

  # Environment variables
  shellHook = ''
    # Rust env vars
    export RUST_BACKTRACE=1
    export RUSTFLAGS="-C target-cpu=native"
    
    # Go env vars
    export GOPATH="$PWD/.go"
    export PATH="$GOPATH/bin:$PATH"
    export GO111MODULE=on
    
    # Project structure setup
    if [ ! -d "frontend" ]; then
      mkdir -p frontend/cmd/nix-timemach
      mkdir -p frontend/internal/{ui,models}
      
      # Initialize Go module
      cd frontend
      go mod init nix-timemach
      go mod tidy
      cd ..
    fi

    if [ ! -d "backend" ]; then
      mkdir -p backend/src/{models,services}
      
      # Initialize Rust project
      cd backend
      cargo init --bin
      cd ..
    fi

    echo "Development environment ready!"
    echo "Frontend (Go): cd frontend"
    echo "Backend (Rust): cd backend"
  '';
}
