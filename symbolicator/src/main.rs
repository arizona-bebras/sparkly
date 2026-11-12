use anyhow::{Context, Result, ensure};
use sparkly_symbolicator::{Symbols, parse_parcel_json, symbolicate};
use std::io::Write;

fn main() -> Result<()> {
    let args: Vec<_> = std::env::args_os().skip(1).collect();
    ensure!(
        args.len() == 2,
        "usage: sparkly-symbolicator <parcel.json> <firmware.elf>"
    );
    let bytes = std::fs::read(&args[0]).context("read protobuf JSON parcel")?;
    ensure!(bytes.len() <= 8 << 20, "JSON parcel exceeds 8 MiB");
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
