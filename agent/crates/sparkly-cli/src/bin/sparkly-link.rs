fn main() {
    let result = sparkly_cli::linker::run(std::env::args_os().skip(1).collect());
    match result {
        Ok(code) => std::process::exit(code),
        Err(error) => {
            eprintln!("sparkly-link: {error}");
            std::process::exit(1);
        }
    }
}
