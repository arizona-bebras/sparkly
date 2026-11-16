use std::{
    collections::{BTreeSet, HashMap},
    future::Future,
    rc::Rc,
};

use crate::{
    Output, Symbols,
    domain::{BatchContext, SymbolicatedBatch},
    symbolicate_optional, wire,
};
use anyhow::{Context, Result, ensure};

/// Asynchronous artifact port. None means no artifact for this build; an error aborts processing for retry.
/// Implementations own fetching, validation, and caching of symbol artifacts.
pub trait SymbolProvider {
    fn symbols_for(&mut self, build_id: &str) -> impl Future<Output = Result<Option<Rc<Symbols>>>>;
}

pub struct SymbolicationTask {
    pub context: BatchContext,
    pub parcel: wire::Parcel,
}

/// Orchestrates build-context changes without knowing where artifacts are stored.
pub struct SymbolicationService<P> {
    provider: P,
}

impl<P: SymbolProvider> SymbolicationService<P> {
    pub fn new(provider: P) -> Self {
        Self { provider }
    }

    pub async fn execute(&mut self, task: SymbolicationTask) -> Result<SymbolicatedBatch> {
        let parcel = task.parcel;
        let mut output = Output {
            producer_batch_id: hex::encode(&parcel.batch_id),
            dropped_records: parcel.dropped_records,
            ..Output::default()
        };
        let mut artifacts = HashMap::new();
        let mut used_builds = BTreeSet::new();
        let mut boot = parcel.initial_boot.clone();
        let mut start = 0;
        while start < parcel.events.len() {
            if let Some(wire::event::Payload::Boot(next)) = &parcel.events[start].payload {
                boot = Some(next.clone());
            }
            let build_id = boot
                .as_ref()
                .map(|b| b.build_id.as_str())
                .filter(|id| !id.is_empty());
            let symbols = if let Some(build_id) = build_id {
                if !artifacts.contains_key(build_id) {
                    let symbols = self
                        .provider
                        .symbols_for(build_id)
                        .await
                        .with_context(|| format!("load symbols for build {build_id}"))?;
                    if let Some(symbols) = &symbols {
                        ensure!(
                            symbols.build_id() == build_id,
                            "symbol provider returned an ELF with a different build ID"
                        );
                    }
                    artifacts.insert(build_id.to_owned(), symbols);
                }
                artifacts.get(build_id).and_then(|s| s.as_deref())
            } else {
                None
            };
            if let Some(symbols) = symbols {
                used_builds.insert(symbols.build_id().to_owned());
            }
            // Each span has a stable boot context. Boot events remain real output rows.
            let end = parcel.events[start + 1..]
                .iter()
                .position(|e| matches!(e.payload, Some(wire::event::Payload::Boot(_))))
                .map(|offset| start + 1 + offset)
                .unwrap_or(parcel.events.len());
            let span = wire::Parcel {
                initial_boot: boot.clone(),
                events: parcel.events[start..end].to_vec(),
                ..wire::Parcel::default()
            };
            append_output(&mut output, symbolicate_optional(&span, symbols), start);
            start = end;
        }
        let symbol_build_ids: Vec<_> = used_builds.into_iter().collect();
        if symbol_build_ids.len() == 1 {
            output.symbol_build_id = symbol_build_ids[0].clone();
        }
        Ok(SymbolicatedBatch {
            context: task.context,
            output,
            symbol_build_ids,
        })
    }
}

fn append_output(out: &mut Output, mut span: Output, offset: usize) {
    for row in &mut span.boots {
        row.common.event_index += offset;
    }
    for row in &mut span.metrics {
        row.common.event_index += offset;
    }
    for row in &mut span.logs {
        row.common.event_index += offset;
    }
    for row in &mut span.crashes {
        row.common.event_index += offset;
    }
    for diagnostic in &mut span.diagnostics {
        if let Some(index) = &mut diagnostic.event_index {
            *index += offset;
        }
    }
    out.boots.extend(span.boots);
    out.metrics.extend(span.metrics);
    out.logs.extend(span.logs);
    out.crashes.extend(span.crashes);
    out.diagnostics.extend(span.diagnostics);
}
