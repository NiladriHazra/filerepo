use anyhow::{Context, Result, anyhow};
use serde::Deserialize;
use url::Url;

#[derive(Debug, Clone)]
pub struct GitHubUrl {
    pub owner: String,
    pub repo: String,
    pub branch: String,
    pub path: String,
}

impl GitHubUrl {
    pub fn parse(url_str: &str) -> Result<Self> {
        let url = Url::parse(url_str).context("Invalid URL format")?;

        if url.host_str() != Some("github.com") {
            return Err(anyhow!("Not a GitHub URL"));
        }

        let path_segments: Vec<&str> = url
            .path_segments()
            .ok_or_else(|| anyhow!("Invalid URL path"))?
            .collect();

        if path_segments.len() < 2 {
            return Err(anyhow!("URL must contain owner and repository"));
        }

        let owner = path_segments[0].to_string();
        let repo = path_segments[1].to_string();

        let (branch, path) = if path_segments.len() >= 4
            && (path_segments[2] == "tree" || path_segments[2] == "blob")
        {
            let branch = path_segments[3].to_string();
            let path = if path_segments.len() > 4 {
                path_segments[4..].join("/")
            } else {
                String::new()
            };
            (branch, path)
        } else {
            ("main".to_string(), String::new())
        };

        Ok(GitHubUrl {
            owner,
            repo,
            branch,
            path,
        })
    }

    pub fn get_local_git_remote() -> Option<String> {
        use std::process::Command;
        let output = Command::new("git")
            .args(["remote", "get-url", "origin"])
            .output()
            .ok()?;

        if output.status.success() {
            let url = String::from_utf8_lossy(&output.stdout).trim().to_string();
            if !url.is_empty() {
                if url.starts_with("git@github.com:") {
                    let path = url
                        .trim_start_matches("git@github.com:")
                        .trim_end_matches(".git");
                    return Some(format!("https://github.com/{}", path));
                }

                if let Ok(parsed) = Url::parse(&url) {
                    if parsed.host_str() == Some("github.com") {
                        return Some(format!(
                            "https://github.com{}",
                            parsed.path().trim_end_matches(".git")
                        ));
                    }
                }
            }
        }
        None
    }

    pub fn api_url(&self) -> String {
        let base = format!(
            "https://api.github.com/repos/{}/{}/contents",
            self.owner, self.repo
        );
        if self.path.is_empty() {
            format!("{}?ref={}", base, self.branch)
        } else {
            format!("{}/{}?ref={}", base, self.path, self.branch)
        }
    }
}

#[derive(Debug, Clone, Deserialize, PartialEq)]
pub struct RepoItem {
    pub name: String,
    #[serde(rename = "type")]
    pub item_type: String,
    pub path: String,
    pub download_url: Option<String>,
    pub url: String,
    pub size: Option<u64>,
    #[serde(skip)]
    pub selected: bool,
    #[serde(skip)]
    pub lfs_oid: Option<String>,
    #[serde(skip)]
    pub lfs_size: Option<u64>,
    #[serde(skip)]
    pub lfs_download_url: Option<String>,
}

impl RepoItem {
    pub fn is_dir(&self) -> bool {
        self.item_type == "dir"
    }

    pub fn is_file(&self) -> bool {
        self.item_type == "file"
    }

    pub fn is_lfs(&self) -> bool {
        self.lfs_oid.is_some()
    }

    pub fn actual_size(&self) -> Option<u64> {
        self.lfs_size.or(self.size)
    }

    pub fn actual_download_url(&self) -> Option<&String> {
        self.lfs_download_url
            .as_ref()
            .or(self.download_url.as_ref())
    }
}

#[derive(Debug, Clone)]
pub struct LfsPointer {
    pub oid: String,
    pub size: u64,
}

impl LfsPointer {
    pub fn parse(content: &str) -> Option<Self> {
        if !content.starts_with("version https://git-lfs.github.com/spec/v1") {
            return None;
        }

        let mut oid = None;
        let mut size = None;

        for line in content.lines() {
            if line.starts_with("oid sha256:") {
                oid = Some(line.trim_start_matches("oid sha256:").to_string());
            } else if line.starts_with("size ") {
                size = line.trim_start_matches("size ").parse().ok();
            }
        }

        match (oid, size) {
            (Some(oid), Some(size)) => Some(LfsPointer { oid, size }),
            _ => None,
        }
    }
}

#[derive(Debug, Deserialize)]
struct LfsBatchResponse {
    objects: Vec<LfsResponseObject>,
}

#[derive(Debug, Deserialize)]
struct LfsResponseObject {
    #[allow(dead_code)]
    oid: String,
    #[allow(dead_code)]
    size: u64,
    actions: Option<LfsActions>,
}

#[derive(Debug, Deserialize)]
struct LfsActions {
    download: Option<LfsDownloadAction>,
}

#[derive(Debug, Deserialize)]
struct LfsDownloadAction {
    href: String,
}

#[derive(Debug, Deserialize)]
#[serde(untagged)]
enum ContentsResponse {
    Many(Vec<RepoItem>),
    One(RepoItem),
}

#[derive(Debug, Deserialize)]
pub struct GitTreeResponse {
    pub tree: Vec<GitTreeEntry>,
    pub truncated: bool,
}

#[derive(Debug, Deserialize)]
pub struct GitTreeEntry {
    pub path: String,
    #[allow(dead_code)]
    pub mode: String,
    #[serde(rename = "type")]
    pub entry_type: String,
    pub size: Option<u64>,
    #[allow(dead_code)]
    pub sha: String,
    pub url: String,
}

#[derive(Debug, thiserror::Error)]
pub enum GitHubError {
    #[error("Invalid token. Falling back to public API.")]
    InvalidToken,
    #[error(
        "Rate limit exceeded for {0}. Consider adding a token via `filerepo config set token YOUR_TOKEN`."
    )]
    RateLimitReached(String),
    #[error("Resource not found: {0}")]
    NotFound(String),
    #[error("API Error: {0}")]
    ApiError(String),
    #[error(transparent)]
    Other(#[from] anyhow::Error),
}

#[derive(Clone)]
pub struct GitHubClient {
    client: reqwest::Client,
    token: Option<String>,
}

impl GitHubClient {
    pub fn new(token: Option<String>) -> Result<Self> {
        let client = reqwest::Client::builder()
            .user_agent("filerepo/1.0.0")
            .build()
            .context("Failed to create HTTP client")?;
        Ok(GitHubClient { client, token })
    }

    async fn request(
        &self,
        method: reqwest::Method,
        url: &str,
        body: Option<serde_json::Value>,
    ) -> std::result::Result<reqwest::Response, GitHubError> {
        let mut builder = self.client.request(method, url);

        if let Some(token) = &self.token {
            builder = builder.header("Authorization", format!("token {}", token));
        }

        if let Some(body) = body {
            builder = builder.json(&body);
        }

        let response = builder
            .send()
            .await
            .map_err(|e| GitHubError::ApiError(e.to_string()))?;

        match response.status().as_u16() {
            200..=299 => Ok(response),
            401 if self.token.is_some() => Err(GitHubError::InvalidToken),
            403 => {
                let remaining = response
                    .headers()
                    .get("X-RateLimit-Remaining")
                    .and_then(|h| h.to_str().ok())
                    .and_then(|s| s.parse::<u32>().ok())
                    .unwrap_or(1);

                if remaining == 0 {
                    let level = if self.token.is_some() {
                        "authenticated user"
                    } else {
                        "unauthenticated user"
                    };
                    Err(GitHubError::RateLimitReached(level.to_string()))
                } else if self.token.is_some() {
                    Err(GitHubError::InvalidToken)
                } else {
                    Err(GitHubError::ApiError(format!(
                        "Forbidden: {}",
                        response.status()
                    )))
                }
            }
            404 => Err(GitHubError::NotFound(url.to_string())),
            _ => Err(GitHubError::ApiError(format!("HTTP {}", response.status()))),
        }
    }

    pub async fn fetch_contents(&self, url: &str) -> Result<Vec<RepoItem>> {
        let response = self.request(reqwest::Method::GET, url, None).await?;

        let items: ContentsResponse = response
            .json()
            .await
            .context("Failed to parse GitHub API response")?;

        Ok(match items {
            ContentsResponse::Many(items) => items,
            ContentsResponse::One(item) => vec![item],
        })
    }

    pub async fn fetch_recursive_tree(
        &self,
        owner: &str,
        repo: &str,
        branch: &str,
    ) -> std::result::Result<GitTreeResponse, GitHubError> {
        let url = format!(
            "https://api.github.com/repos/{}/{}/git/trees/{}?recursive=1",
            owner, repo, branch
        );
        let response = self.request(reqwest::Method::GET, &url, None).await?;

        let tree: GitTreeResponse = response
            .json()
            .await
            .map_err(|e| GitHubError::ApiError(e.to_string()))?;
        Ok(tree)
    }

    pub async fn fetch_raw_content(&self, url: &str) -> Result<String> {
        let response = self.request(reqwest::Method::GET, url, None).await?;
        let content = response.text().await.context("Failed to read content")?;
        Ok(content)
    }

    pub async fn download_binary(&self, url: &str) -> Result<Vec<u8>> {
        let response = self.request(reqwest::Method::GET, url, None).await?;
        let bytes = response
            .bytes()
            .await
            .context("Failed to read binary content")?;
        Ok(bytes.to_vec())
    }

    pub async fn get_lfs_download_url(
        &self,
        owner: &str,
        repo: &str,
        oid: &str,
        size: u64,
    ) -> Result<String> {
        let batch_url = format!(
            "https://github.com/{}/{}.git/info/lfs/objects/batch",
            owner, repo
        );

        let request_body = serde_json::json!({
            "operation": "download",
            "transfers": ["basic"],
            "objects": [{ "oid": oid, "size": size }]
        });

        let response = self
            .request(reqwest::Method::POST, &batch_url, Some(request_body))
            .await?;

        let batch_response: LfsBatchResponse = response
            .json()
            .await
            .context("Failed to parse LFS response")?;

        batch_response
            .objects
            .into_iter()
            .next()
            .and_then(|obj| obj.actions)
            .and_then(|actions| actions.download)
            .map(|download| download.href)
            .ok_or_else(|| anyhow!("No download URL in LFS response"))
    }

    pub async fn resolve_lfs_files(
        &self,
        items: &mut [RepoItem],
        owner: &str,
        repo: &str,
        branch: &str,
    ) {
        for item in items.iter_mut() {
            if item.is_file() {
                if let Some(size) = item.size {
                    if size < 1024 {
                        if let Some(download_url) = &item.download_url {
                            if let Ok(content) = self.fetch_raw_content(download_url).await {
                                if let Some(pointer) = LfsPointer::parse(&content) {
                                    item.lfs_oid = Some(pointer.oid.clone());
                                    item.lfs_size = Some(pointer.size);

                                    if let Ok(lfs_url) = self
                                        .get_lfs_download_url(
                                            owner,
                                            repo,
                                            &pointer.oid,
                                            pointer.size,
                                        )
                                        .await
                                    {
                                        item.lfs_download_url = Some(lfs_url);
                                    } else {
                                        let media_url = format!(
                                            "https://media.githubusercontent.com/media/{}/{}/{}/{}",
                                            owner, repo, branch, item.path
                                        );
                                        item.lfs_download_url = Some(media_url);
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_github_url() {
        let url = "https://github.com/rust-lang/rust/tree/master/src/tools";
        let parsed = GitHubUrl::parse(url).unwrap();
        assert_eq!(parsed.owner, "rust-lang");
        assert_eq!(parsed.repo, "rust");
        assert_eq!(parsed.branch, "master");
        assert_eq!(parsed.path, "src/tools");
    }

    #[test]
    fn test_parse_root_url() {
        let url = "https://github.com/rust-lang/rust";
        let parsed = GitHubUrl::parse(url).unwrap();
        assert_eq!(parsed.owner, "rust-lang");
        assert_eq!(parsed.repo, "rust");
        assert_eq!(parsed.branch, "main");
        assert_eq!(parsed.path, "");
    }

    #[test]
    fn test_parse_blob_url() {
        let url = "https://github.com/owner/repo/blob/main/src/lib.rs";
        let parsed = GitHubUrl::parse(url).unwrap();
        assert_eq!(parsed.path, "src/lib.rs");
    }

    #[test]
    fn test_parse_invalid_non_github_url() {
        assert!(GitHubUrl::parse("https://gitlab.com/user/repo").is_err());
    }

    #[test]
    fn test_parse_invalid_no_repo() {
        assert!(GitHubUrl::parse("https://github.com/owner").is_err());
    }

    #[test]
    fn test_api_url_with_path() {
        let gh = GitHubUrl {
            owner: "owner".into(),
            repo: "repo".into(),
            branch: "main".into(),
            path: "src/lib.rs".into(),
        };
        assert_eq!(
            gh.api_url(),
            "https://api.github.com/repos/owner/repo/contents/src/lib.rs?ref=main"
        );
    }

    #[test]
    fn test_lfs_pointer_parse_valid() {
        let content =
            "version https://git-lfs.github.com/spec/v1\noid sha256:abc123def456\nsize 12345";
        let pointer = LfsPointer::parse(content).unwrap();
        assert_eq!(pointer.oid, "abc123def456");
        assert_eq!(pointer.size, 12345);
    }

    #[test]
    fn test_lfs_pointer_parse_not_lfs() {
        assert!(LfsPointer::parse("regular file content").is_none());
    }

    #[test]
    fn test_parse_root_url_from_git_remote() {
        let url = Url::parse("https://github.com/owner/repo.git").unwrap();
        assert_eq!(
            format!("https://github.com{}", url.path().trim_end_matches(".git")),
            "https://github.com/owner/repo"
        );
    }
}
