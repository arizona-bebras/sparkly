use clap::{Parser, Subcommand};
use sparkly_cli::{
    Result,
    init::{InitOptions, initialize},
};
use std::path::PathBuf;

#[derive(Parser)]
#[command(version, about = "Sparkly firmware setup and ELF tools")]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Configure an existing Rust firmware project (currently ESP32-S3).
    Init(InitOptions),
    /// Validate an ELF's build IDs and print its metric dictionary.
    Inspect { elf: PathBuf },
    /// Finalize a manually linked ELF before image conversion or signing.
    Finalize { elf: PathBuf },
}

fn run() -> Result<()> {
    match Cli::parse().command {
        Command::Init(options) => {
            let root = initialize(&options)?;
            println!(
                "Configured {}. Install sparkly-link on PATH, then use cargo +esp build.",
                root.display()
            );
        }
        Command::Inspect { elf } => print_artifact(&sparkly_build::inspect_elf(elf)?),
        Command::Finalize { elf } => {
            let result = sparkly_build::finalize_elf(elf)?;
            println!(
                "{}",
                if result.changed {
                    "Finalized runtime build ID."
                } else {
                    "Already finalized."
                }
            );
            print_artifact(&result.artifact);
        }
    }
    Ok(())
}

fn print_artifact(artifact: &sparkly_build::Artifact) {
    print!("build_id: ");
    for byte in &artifact.build_id {
        print!("{byte:02x}");
    }
    println!(
        "\nruntime_id: {}",
        if artifact.runtime_id.is_some() {
            "verified"
        } else {
            "not present"
        }
    );
    for metric in &artifact.metrics {
        println!(
            "{:016x} {} {}",
            metric.id,
            if metric.kind == 1 {
                "gauge"
            } else {
                "counter_delta"
            },
            metric.name
        );
    }
}

fn main() -> std::process::ExitCode {
    match run() {
        Ok(()) => std::process::ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("sparkly: {error}");
            std::process::ExitCode::FAILURE
        }
    }
}
