use crate::github::{GitHubClient, RepoItem};
use anyhow::{Context, Result};
use futures::stream::{self, StreamExt};
use std::path::PathBuf;
use std::sync::Arc;
use std::sync::atomic::{AtomicUsize, Ordering};
use tokio::fs;

const MAX_CONCURRENT_DOWNLOADS: usize = 8;

pub struct DownloadProgress {
    pub total: usize,
    pub completed: AtomicUsize,
    pub current_file: tokio::sync::Mutex<String>,
}

pub struct Downloader {
    client: GitHubClient,
    base_path: PathBuf,
}

impl Downloader {
    pub fn new(base_path: PathBuf, token: Option<String>) -> Result<Self> {
        std::fs::create_dir_all(&base_path).context("Failed to create download directory")?;
        Ok(Downloader {
            client: GitHubClient::new(token)?,
            base_path,
        })
    }

    pub async fn download_items(
        &self,
        items: &[RepoItem],
        progress: Arc<DownloadProgress>,
    ) -> Result<Vec<String>> {
        let errors = Arc::new(tokio::sync::Mutex::new(Vec::new()));
        let client = self.client.clone();
        let base_path = self.base_path.clone();
        let errors_for_tasks = errors.clone();

        stream::iter(items.iter().cloned())
            .for_each_concurrent(MAX_CONCURRENT_DOWNLOADS, move |item| {
                let client = client.clone();
                let base = base_path.clone();
                let progress = progress.clone();
                let errors = errors_for_tasks.clone();
                async move {
                    let dest_path = base.join(&item.name);
                    if let Err(e) = Self::download_file(&client, &item, dest_path, &progress).await
                    {
                        errors
                            .lock()
                            .await
                            .push(format!("Failed to download {}: {}", item.name, e));
                    }
                    progress.completed.fetch_add(1, Ordering::Relaxed);
                }
            })
            .await;

        let result = errors.lock().await.clone();
        Ok(result)
    }

    async fn download_file(
        client: &GitHubClient,
        item: &RepoItem,
        dest_path: PathBuf,
        progress: &DownloadProgress,
    ) -> Result<()> {
        let download_url = item
            .actual_download_url()
            .context("No download URL for file")?;

        {
            let mut current = progress.current_file.lock().await;
            *current = item.name.clone();
        }

        let content = client
            .download_binary(download_url)
            .await
            .context("Failed to download file")?;

        if let Some(parent) = dest_path.parent() {
            fs::create_dir_all(parent)
                .await
                .context("Failed to create parent download directory")?;
        }

        fs::write(&dest_path, content)
            .await
            .context(format!("Failed to write file: {:?}", dest_path))?;

        Ok(())
    }
}
