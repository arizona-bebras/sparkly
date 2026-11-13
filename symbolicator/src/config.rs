use anyhow::{Context, Result};

// Temporary symbol source: every task uses this one local ELF, loaded at startup.
pub const FIXED_ELF_PATH: &str = env!("FIXED_ELF_PATH");

pub struct Config {
    pub rabbitmq_url: String,
}

impl Config {
    pub fn from_env() -> Result<Self> {
        let rabbitmq_url = match std::env::var("RABBITMQ_URL") {
            Ok(value) if !value.is_empty() => value,
            Ok(_) | Err(std::env::VarError::NotPresent) => "amqp://localhost:5672/".into(),
            Err(error) => return Err(error).context("read RABBITMQ_URL"),
        };
        Ok(Self { rabbitmq_url })
    }
}
