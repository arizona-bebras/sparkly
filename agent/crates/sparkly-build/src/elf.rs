use crate::{MetricDefinition, Result, metrics::read_metrics};
use object::{Object, ObjectSection, ObjectSegment, ObjectSymbol, SectionFlags};
use std::{fs, io::Write, ops::Range, path::Path};

const SLOT_SYMBOL: &str = "__sparkly_build_id";
const SLOT_SIZE: usize = 40;

#[derive(Debug)]
pub struct Artifact {
    pub build_id: Vec<u8>,
    /// None for an ELF that does not use Sparkly's runtime build-ID API.
    pub runtime_id: Option<Vec<u8>>,
    pub metrics: Vec<MetricDefinition>,
}

#[derive(Debug)]
pub struct FinalizedArtifact {
    pub artifact: Artifact,
    pub changed: bool,
}

fn parse(bytes: &[u8]) -> Result<object::File<'_>> {
    let elf = object::File::parse(bytes)?;
    if elf.format() != object::BinaryFormat::Elf {
        return Err("expected an ELF artifact".into());
    }
    Ok(elf)
}

fn slot_range(elf: &object::File<'_>, file_len: usize) -> Result<Option<Range<usize>>> {
    let mut symbols = elf
        .symbols()
        .filter(|symbol| symbol.name() == Ok(SLOT_SYMBOL) && symbol.is_definition());
    let Some(symbol) = symbols.next() else {
        return Ok(None);
    };
    if symbols.next().is_some() || symbol.size() != SLOT_SIZE as u64 {
        return Err("ambiguous or incorrectly sized runtime build-ID slot".into());
    }
    let section = elf.section_by_index(
        symbol
            .section_index()
            .ok_or("build-ID slot has no section")?,
    )?;
    if !matches!(section.flags(), SectionFlags::Elf { sh_flags }
        if sh_flags & object::elf::SHF_ALLOC as u64 != 0 && sh_flags & object::elf::SHF_WRITE as u64 == 0)
    {
        return Err("runtime build-ID slot must be allocated read-only data".into());
    }
    let (offset, len) = section
        .file_range()
        .ok_or("build-ID slot has no file data")?;
    let relative = symbol
        .address()
        .checked_sub(section.address())
        .ok_or("invalid build-ID address")?;
    if relative
        .checked_add(SLOT_SIZE as u64)
        .is_none_or(|end| end > len)
    {
        return Err("build-ID slot exceeds its section".into());
    }
    if !elf.segments().any(|segment| {
        segment
            .data_range(symbol.address(), SLOT_SIZE as u64)
            .ok()
            .flatten()
            .is_some()
    }) {
        return Err("runtime build-ID slot is not in a loadable segment".into());
    }
    let start = usize::try_from(
        offset
            .checked_add(relative)
            .ok_or("invalid build-ID offset")?,
    )?;
    let end = start
        .checked_add(SLOT_SIZE)
        .ok_or("invalid build-ID range")?;
    if end > file_len {
        return Err("truncated build-ID slot".into());
    }
    Ok(Some(start..end))
}

fn validate_header(slot: &[u8]) -> Result<()> {
    if slot.len() != SLOT_SIZE || &slot[..4] != b"SPID" || slot[4] != 1 || slot[6..8] != [0, 0] {
        return Err("invalid runtime build-ID slot header".into());
    }
    Ok(())
}

fn gnu_build_id<'a>(elf: &object::File<'a>) -> Result<&'a [u8]> {
    // GNU ld emits an INFO note as PROGBITS, while LLD preserves SHT_NOTE.
    // Read the named section in either case; both contain the GNU note format.
    if let Some(section) = elf.section_by_name(".note.gnu.build-id") {
        let note = section.data()?;
        let integer = |bytes: &[u8]| -> u32 {
            let bytes = bytes.try_into().unwrap();
            if elf.is_little_endian() {
                u32::from_le_bytes(bytes)
            } else {
                u32::from_be_bytes(bytes)
            }
        };
        if note.len() < 16
            || integer(&note[..4]) != 4
            || integer(&note[8..12]) != 3
            || &note[12..16] != b"GNU\0"
        {
            return Err("invalid GNU build-ID note".into());
        }
        let len = integer(&note[4..8]) as usize;
        if !(1..=32).contains(&len) {
            return Err("unsupported GNU build-ID length".into());
        }
        return note
            .get(16..16 + len)
            .ok_or_else(|| "truncated GNU build-ID note".into());
    }
    elf.build_id()?
        .ok_or_else(|| "missing GNU build ID; use sparkly-link".into())
}

fn inspect(bytes: &[u8]) -> Result<Artifact> {
    let elf = parse(bytes)?;
    let build_id = gnu_build_id(&elf)?.to_vec();
    if !(1..=32).contains(&build_id.len()) {
        return Err("unsupported GNU build-ID length".into());
    }
    let runtime_id = if let Some(range) = slot_range(&elf, bytes.len())? {
        let slot = &bytes[range];
        validate_header(slot)?;
        let len = usize::from(slot[5]);
        if len == 0 {
            return Err(
                "runtime build ID is not finalized; use sparkly-link or sparkly finalize".into(),
            );
        }
        if len != build_id.len()
            || slot[8..8 + len] != build_id
            || slot[8 + len..].iter().any(|b| *b != 0)
        {
            return Err("runtime build ID does not match GNU build ID".into());
        }
        Some(slot[8..8 + len].to_vec())
    } else {
        None
    };
    Ok(Artifact {
        build_id,
        runtime_id,
        metrics: read_metrics(&elf)?,
    })
}

pub fn inspect_elf(path: impl AsRef<Path>) -> Result<Artifact> {
    inspect(&fs::read(path)?)
}

/// Used by the wrapper to pass through uninstrumented executables and libraries.
pub fn needs_finalization(path: impl AsRef<Path>) -> Result<bool> {
    let bytes = fs::read(path)?;
    let elf = parse(&bytes)?;
    Ok(slot_range(&elf, bytes.len())?.is_some())
}

/// Copies the GNU note's descriptor into the reserved runtime slot. No sizes,
/// addresses, instructions, note bytes, or unrelated ELF contents are changed.
/// Re-running on an already finalized ELF is idempotent; conflicting IDs fail.
/// Must run before stripping, image conversion, compression or signing.
pub fn finalize_elf(path: impl AsRef<Path>) -> Result<FinalizedArtifact> {
    let path = path.as_ref();
    let mut bytes = fs::read(path)?;
    let elf = parse(&bytes)?;
    let id = gnu_build_id(&elf)?.to_vec();
    if !(1..=32).contains(&id.len()) {
        return Err("unsupported GNU build-ID length".into());
    }
    let range = slot_range(&elf, bytes.len())?.ok_or("missing runtime build-ID slot")?;
    read_metrics(&elf)?;
    let slot = &bytes[range.clone()];
    validate_header(slot)?;
    if slot[5] != 0 {
        return Ok(FinalizedArtifact {
            artifact: inspect(&bytes)?,
            changed: false,
        });
    }
    if slot[8..].iter().any(|byte| *byte != 0) {
        return Err("corrupt build-ID placeholder".into());
    }
    bytes[range.start + 5] = id.len() as u8;
    bytes[range.start + 8..range.start + 8 + id.len()].copy_from_slice(&id);
    let artifact = inspect(&bytes)?;

    // Validate everything before replacing the linker's output, preserving its
    // executable permissions. Same-directory persist uses an atomic rename.
    let parent = path
        .parent()
        .filter(|p| !p.as_os_str().is_empty())
        .unwrap_or(Path::new("."));
    let mut temporary = tempfile::NamedTempFile::new_in(parent)?;
    temporary
        .as_file()
        .set_permissions(fs::metadata(path)?.permissions())?;
    temporary.write_all(&bytes)?;
    temporary.as_file().sync_all()?;
    temporary.persist(path).map_err(|error| error.error)?;
    Ok(FinalizedArtifact {
        artifact,
        changed: true,
    })
}
