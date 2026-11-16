use crate::{Symbols, application::SymbolProvider};
use anyhow::{Context, Result};
use std::{path::Path, rc::Rc};

/// Temporary symbol source. Only an exact GNU build ID match can use this ELF.
pub struct FixedElfProvider {
    symbols: Rc<Symbols>,
}

impl FixedElfProvider {
    pub fn load(path: impl AsRef<Path>) -> Result<Self> {
        let symbols = Symbols::load(path).context("load fixed symbol ELF")?;
        Ok(Self {
            symbols: Rc::new(symbols),
        })
    }
}

impl SymbolProvider for FixedElfProvider {
    async fn symbols_for(&mut self, build_id: &str) -> Result<Option<Rc<Symbols>>> {
        Ok((self.symbols.build_id() == build_id).then(|| Rc::clone(&self.symbols)))
    }
}
