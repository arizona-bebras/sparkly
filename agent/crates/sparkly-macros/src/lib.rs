//! Compile-time metric IDs and a versioned, non-allocated ELF dictionary.
use proc_macro::TokenStream;
use proc_macro_crate::{FoundCrate, crate_name};
use quote::{format_ident, quote};
use syn::{LitStr, parse_macro_input};

#[proc_macro]
pub fn metric(input: TokenStream) -> TokenStream {
    expand(parse_macro_input!(input as LitStr), 1, "Gauge")
}

#[proc_macro]
pub fn counter(input: TokenStream) -> TokenStream {
    expand(parse_macro_input!(input as LitStr), 2, "Counter")
}

fn expand(name: LitStr, kind: u8, handle: &str) -> TokenStream {
    let value = name.value();
    if value.is_empty() || value.len() > 255 || value.bytes().any(|b| b.is_ascii_control()) {
        return syn::Error::new(
            name.span(),
            "metric names must contain 1..=255 bytes and no control characters",
        )
        .to_compile_error()
        .into();
    }
    // FNV-1a 64, over the exact UTF-8 spelling. This is an identifier, not a security hash.
    let id = value.bytes().fold(0xcbf29ce484222325_u64, |hash, byte| {
        (hash ^ u64::from(byte)).wrapping_mul(0x100000001b3)
    });
    // Wire layout: magic(4), version(1), kind(1), name_len(2 LE), id(8 LE), UTF-8 name.
    // No pointers or Rust-layout-dependent structs appear in this section.
    let mut metadata = b"SPKM".to_vec();
    metadata.extend_from_slice(&[1, kind]);
    metadata.extend_from_slice(&(value.len() as u16).to_le_bytes());
    metadata.extend_from_slice(&id.to_le_bytes());
    metadata.extend_from_slice(value.as_bytes());
    let len = metadata.len();
    let root = match crate_name("sparkly-agent") {
        Ok(FoundCrate::Name(name)) => {
            let ident = format_ident!("{name}");
            quote!(::#ident)
        }
        // The library declares an alias, also valid in its own integration tests/examples.
        Ok(FoundCrate::Itself) => quote!(::sparkly_agent),
        Err(error) => {
            return syn::Error::new(name.span(), error.to_string())
                .to_compile_error()
                .into();
        }
    };
    let handle = format_ident!("{handle}");
    quote!({
        #[used]
        #[unsafe(link_section = ".sparkly.metrics")]
        static METADATA: [u8; #len] = [#(#metadata),*];
        #root::#handle::new(#id)
    })
    .into()
}
