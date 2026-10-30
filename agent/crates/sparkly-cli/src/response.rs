//! Inspect GNU response files without altering the arguments passed to the real
//! linker. No shell expansion or command execution is performed.
use crate::Result;
use std::{
    ffi::{OsStr, OsString},
    fs,
    path::PathBuf,
};

pub(crate) fn expand(arguments: &[OsString]) -> Result<Vec<OsString>> {
    let mut result = Vec::new();
    expand_into(arguments, &mut result, 0)?;
    Ok(result)
}

fn expand_into(arguments: &[OsString], output: &mut Vec<OsString>, depth: usize) -> Result<()> {
    if depth > 16 {
        return Err("response-file nesting exceeds 16 levels (possible cycle)".into());
    }
    for argument in arguments {
        if let Some(path) = response_path(argument) {
            let bytes = fs::read(path)?;
            let bytes = if bytes.starts_with(&[0xff, 0xfe]) {
                if !(bytes.len() - 2).is_multiple_of(2) {
                    return Err("invalid UTF-16 response file".into());
                }
                String::from_utf16(
                    &bytes[2..]
                        .as_chunks::<2>()
                        .0
                        .iter()
                        .map(|c| u16::from_le_bytes([c[0], c[1]]))
                        .collect::<Vec<_>>(),
                )?
                .into_bytes()
            } else {
                bytes
            };
            expand_into(&tokenize(&bytes)?, output, depth + 1)?;
        } else {
            output.push(argument.clone());
        }
    }
    Ok(())
}

fn response_path(argument: &OsStr) -> Option<PathBuf> {
    #[cfg(unix)]
    {
        use std::os::unix::ffi::{OsStrExt, OsStringExt};
        argument
            .as_bytes()
            .strip_prefix(b"@")
            .map(|path| PathBuf::from(OsString::from_vec(path.to_vec())))
    }
    #[cfg(not(unix))]
    {
        argument.to_str()?.strip_prefix('@').map(PathBuf::from)
    }
}

fn tokenize(bytes: &[u8]) -> Result<Vec<OsString>> {
    let mut result = Vec::new();
    let mut word = Vec::new();
    let mut quote = None;
    let mut escaped = false;
    let mut started = false;
    for &byte in bytes {
        if byte == 0 {
            return Err("NUL byte in response file".into());
        }
        if escaped {
            word.push(byte);
            escaped = false;
            started = true;
            continue;
        }
        if byte == b'\\' {
            escaped = true;
            started = true;
            continue;
        }
        if let Some(delimiter) = quote {
            if byte == delimiter {
                quote = None;
            } else {
                word.push(byte);
            }
        } else if byte == b'\'' || byte == b'"' {
            quote = Some(byte);
            started = true;
        } else if byte.is_ascii_whitespace() {
            if started {
                result.push(os_string(std::mem::take(&mut word))?);
                started = false;
            }
        } else {
            word.push(byte);
            started = true;
        }
    }
    if escaped || quote.is_some() {
        return Err("unterminated escape/quote in response file".into());
    }
    if started {
        result.push(os_string(word)?);
    }
    Ok(result)
}

fn os_string(bytes: Vec<u8>) -> Result<OsString> {
    #[cfg(unix)]
    {
        use std::os::unix::ffi::OsStringExt;
        Ok(OsString::from_vec(bytes))
    }
    #[cfg(not(unix))]
    {
        Ok(String::from_utf8(bytes)?.into())
    }
}

/// rustc normally uses `-o path`; accept other GNU spellings as well.
pub(crate) fn output_path(arguments: &[OsString]) -> Result<Option<OsString>> {
    let mut output = None;
    let mut arguments = arguments.iter();
    while let Some(argument) = arguments.next() {
        if argument == OsStr::new("-o") || argument == OsStr::new("--output") {
            output = Some(
                arguments
                    .next()
                    .ok_or("missing linker output path")?
                    .clone(),
            );
        } else if let Some(text) = argument.to_str()
            && let Some(value) = text
                .strip_prefix("--output=")
                .or_else(|| text.strip_prefix("-o"))
            && !value.is_empty()
        {
            output = Some(value.into());
        }
    }
    Ok(output)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn parses_spaces_quotes_and_escaped_paths_without_shell_expansion() {
        let args =
            tokenize(br#"input.o -o 'an output.elf' "dollar$and;literal" escaped\ path"#).unwrap();
        assert_eq!(output_path(&args).unwrap(), Some("an output.elf".into()));
        assert_eq!(args[3], "dollar$and;literal");
        assert_eq!(args[4], "escaped path");
        assert!(tokenize(b"'unfinished").is_err());
    }

    #[test]
    fn follows_nested_response_files_and_rejects_cycles() {
        let dir = tempfile::tempdir().unwrap();
        let inner = dir.path().join("inner.rsp");
        let outer = dir.path().join("outer.rsp");
        fs::write(&inner, "-o 'nested output.elf'").unwrap();
        fs::write(&outer, format!("@{}", inner.display())).unwrap();
        let args = [format!("@{}", outer.display()).into()];
        assert_eq!(
            output_path(&expand(&args).unwrap()).unwrap(),
            Some("nested output.elf".into())
        );
        fs::write(&inner, format!("@{}", outer.display())).unwrap();
        assert!(
            expand(&args)
                .unwrap_err()
                .to_string()
                .contains("possible cycle")
        );
    }
}
