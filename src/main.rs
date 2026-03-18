#![forbid(unsafe_code)]

use anyhow::Result;
use clap::{Parser, Subcommand};
use filerepo::config::Config;
use filerepo::ui;

#[derive(Parser)]
#[command(
    name = "filerepo",
    version,
    about = "A beautiful TUI to browse and download files from GitHub repos"
)]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
    url: Option<String>,
    #[arg(long)]
    cwd: bool,
    #[arg(long)]
    no_folder: bool,
    #[arg(long)]
    token: Option<String>,
}

#[derive(Subcommand)]
enum Commands {
    Config {
        #[command(subcommand)]
        action: ConfigCommand,
    },
}

#[derive(Subcommand)]
enum ConfigCommand {
    Set {
        #[command(subcommand)]
        target: SetTarget,
    },
    Unset {
        #[command(subcommand)]
        target: UnsetTarget,
    },
    List,
}

#[derive(Subcommand)]
enum SetTarget {
    Token { value: String },
    Path { value: String },
}

#[derive(Subcommand)]
enum UnsetTarget {
    Token,
    Path,
}

#[tokio::main]
async fn main() -> Result<()> {
    run(Cli::parse()).await
}

async fn run(cli: Cli) -> Result<()> {
    match cli.command {
        Some(Commands::Config { action }) => handle_config(action),
        None => launch_ui(cli).await,
    }
}

fn handle_config(action: ConfigCommand) -> Result<()> {
    match action {
        ConfigCommand::Set { target } => match target {
            SetTarget::Token { value } => set_config_value(move |config| {
                config.github_token = Some(value);
                "[+] GitHub token saved."
            }),
            SetTarget::Path { value } => {
                Config::validate_path(&value)?;
                set_config_value(move |config| {
                    config.download_path = Some(value);
                    "[+] Download path saved."
                })
            }
        },
        ConfigCommand::Unset { target } => match target {
            UnsetTarget::Token => set_config_value(|config| {
                config.github_token = None;
                "[+] GitHub token removed."
            }),
            UnsetTarget::Path => set_config_value(|config| {
                config.download_path = None;
                "[+] Download path removed."
            }),
        },
        ConfigCommand::List => {
            print_config(&Config::load().unwrap_or_default());
            Ok(())
        }
    }
}

fn set_config_value<F>(update: F) -> Result<()>
where
    F: FnOnce(&mut Config) -> &'static str,
{
    let mut config = Config::load()?;
    let message = update(&mut config);
    config.save()?;
    println!("{message}");
    Ok(())
}

fn print_config(config: &Config) {
    println!("--- filerepo config ---");
    match &config.github_token {
        Some(token) => println!("  Token:         {}", mask_token(token)),
        None => println!("  Token:         (not set)"),
    }

    match &config.download_path {
        Some(path) => println!("  Download Path: {path}"),
        None => println!("  Download Path: (default current working directory)"),
    }
}

fn mask_token(token: &str) -> String {
    if token.len() > 8 {
        format!("{}...{}", &token[..4], &token[token.len() - 4..])
    } else {
        "********".to_string()
    }
}

async fn launch_ui(cli: Cli) -> Result<()> {
    let config = Config::load().unwrap_or_default();
    let token = cli.token.or(config.github_token);
    ui::run_tui(cli.url, token, config.download_path, cli.cwd, cli.no_folder).await
}
