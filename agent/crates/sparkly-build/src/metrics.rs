use crate::Result;
use object::{Object, ObjectSection, SectionFlags};
use std::collections::BTreeMap;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MetricDefinition {
    pub id: u64,
    pub name: String,
    /// 1 = gauge, 2 = counter delta.
    pub kind: u8,
}

pub(crate) fn read_metrics(elf: &object::File<'_>) -> Result<Vec<MetricDefinition>> {
    let Some(section) = elf.section_by_name(".sparkly.metrics") else {
        return Ok(Vec::new());
    };
    if matches!(section.flags(), SectionFlags::Elf { sh_flags } if sh_flags & object::elf::SHF_ALLOC as u64 != 0)
    {
        return Err("metric dictionary is allocated; use sparkly-link".into());
    }
    parse(section.data()?)
}

fn parse(mut bytes: &[u8]) -> Result<Vec<MetricDefinition>> {
    let mut metrics = BTreeMap::new();
    while !bytes.is_empty() {
        if bytes.len() < 16 || &bytes[..4] != b"SPKM" || bytes[4] != 1 {
            return Err("invalid metric dictionary record".into());
        }
        let kind = bytes[5];
        if !matches!(kind, 1 | 2) {
            return Err("unknown metric kind".into());
        }
        let len = u16::from_le_bytes(bytes[6..8].try_into()?) as usize;
        let id = u64::from_le_bytes(bytes[8..16].try_into()?);
        let name = std::str::from_utf8(bytes.get(16..16 + len).ok_or("truncated metric name")?)?;
        if !(1..=255).contains(&len) || name.bytes().any(|b| b.is_ascii_control()) {
            return Err("invalid metric name".into());
        }
        let expected = name.bytes().fold(0xcbf29ce484222325_u64, |hash, byte| {
            (hash ^ u64::from(byte)).wrapping_mul(0x100000001b3)
        });
        if id != expected {
            return Err("metric ID does not match name".into());
        }
        let definition = MetricDefinition {
            id,
            name: name.to_owned(),
            kind,
        };
        if let Some(previous) = metrics.insert(id, definition.clone())
            && previous != definition
        {
            return Err(format!("metric collision/conflicting kind for {name}").into());
        }
        bytes = &bytes[16 + len..];
    }
    Ok(metrics.into_values().collect())
}
