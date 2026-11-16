use anyhow::{Context, Result, ensure};
use sparkly_symbolicator::{
    Symbols,
    application::SymbolicationService,
    config::Config,
    infrastructure::{
        amqp::{AmqpWorker, QueueNames},
        fixed_elf::FixedElfProvider,
        messages::MAX_REQUEST_BYTES,
    },
    parse_parcel_json, symbolicate,
};
use std::{io::Write, path::Path};

#[tokio::main]
async fn main() -> Result<()> {
    let args: Vec<_> = std::env::args_os().skip(1).collect();
    if args.is_empty() || (args.len() == 2 && args[0] == "worker") {
        return run_worker(&args[1]).await;
    }
    ensure!(
        args.len() == 2,
        "usage: sparkly-symbolicator [worker <firmware.elf> | <parcel.json> <firmware.elf>]"
    );
    let bytes = std::fs::read(&args[0]).context("read protobuf JSON parcel")?;
    ensure!(
        bytes.len() <= MAX_REQUEST_BYTES,
        "JSON parcel exceeds 8 MiB"
    );
    let parcel = parse_parcel_json(&bytes).context("decode protobuf JSON parcel")?;
    let symbols = Symbols::load(&args[1]).with_context(|| {
        format!(
            "load symbol ELF {}",
            std::path::Path::new(&args[1]).display()
        )
    })?;
    let output = symbolicate(&parcel, &symbols);
    let mut stdout = std::io::stdout().lock();
    serde_json::to_writer_pretty(&mut stdout, &output)?;
    writeln!(stdout)?;
    Ok(())
}

async fn run_worker(symbols: impl AsRef<Path>) -> Result<()> {
    let config = Config::from_env()?;
    let provider = FixedElfProvider::load(symbols)?;
    let mut service = SymbolicationService::new(provider);
    let worker = AmqpWorker::connect(&config.rabbitmq_url, QueueNames::default()).await?;
    eprintln!("symbolicator consuming symbolication_requests");
    let processing = worker.run(&mut service, shutdown_signal()).await;
    let closing = worker.close().await;
    processing.and(closing)
}

async fn shutdown_signal() -> Result<()> {
    #[cfg(unix)]
    {
        let mut terminate =
            tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
                .context("listen for SIGTERM")?;
        tokio::select! {
         result = tokio::signal::ctrl_c() => result.context("listen for Ctrl-C")?,
         _ = terminate.recv() => {},
        }
    }
    #[cfg(not(unix))]
    tokio::signal::ctrl_c().await.context("listen for Ctrl-C")?;
    Ok(())
}
