use crate::ui::components::toast::{Toast, ToastType};
use anyhow::{Context, Result};
use crossterm::{
    event::{self, Event, KeyCode, KeyEvent, KeyModifiers},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{backend::CrosstermBackend, Terminal};
use std::collections::{HashMap, HashSet};
use std::io;
use std::sync::atomic::Ordering;
use std::sync::Arc;
use tokio::sync::Mutex;

use crate::download::DownloadProgress;
use crate::github::{GitHubClient, GitHubError, GitHubUrl, RepoItem};

pub mod components;
pub mod theme;

const LIST_VISIBLE_HEIGHT: usize = 10;
const PAGE_STEP: usize = 10;

fn install_panic_hook() {
    let original_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(move |panic_info| {
        let _ = disable_raw_mode();
        let _ = execute!(io::stdout(), LeaveAlternateScreen);
        original_hook(panic_info);
    }));
}

#[derive(Debug, PartialEq, Eq)]
pub enum AppMode {
    Input,
    Searching,
    Browse,
}

enum BackNavigation {
    ExitToInput,
    Local {
        items: Vec<RepoItem>,
        url: GitHubUrl,
        cursor: usize,
    },
    Remote {
        url: GitHubUrl,
        cursor: usize,
    },
}

enum EnterDirectory {
    None,
    Local {
        items: Vec<RepoItem>,
        url: GitHubUrl,
    },
    Remote {
        url: GitHubUrl,
    },
}

enum DownloadOutcome {
    Empty,
    Completed {
        download_dir: std::path::PathBuf,
        errors: Vec<String>,
    },
}

pub struct AppState {
    pub mode: AppMode,
    pub url_input: String,
    pub url_cursor: usize,
    pub current_url: Option<GitHubUrl>,
    pub items: Vec<RepoItem>,
    pub cursor: usize,
    pub scroll_offset: usize,
    pub status_message: String,
    pub downloading: bool,
    pub download_progress: Option<Arc<DownloadProgress>>,
    pub navigation_stack: Vec<(GitHubUrl, usize)>,
    pub frame_count: u64,
    pub toast: Option<Toast>,
    pub ascii_mode: bool,
    pub github_token: Option<String>,
    pub download_path: Option<String>,
    pub full_tree: Option<Vec<RepoItem>>,
    pub folder_sizes: HashMap<String, u64>,
    pub cwd: bool,
    pub no_folder: bool,
    pub is_searching: bool,
    pub search_query: String,
    pub selected_paths: HashSet<String>,
}

impl Default for AppState {
    fn default() -> Self {
        Self::new()
    }
}

impl AppState {
    pub fn new() -> Self {
        AppState {
            mode: AppMode::Input,
            url_input: String::new(),
            url_cursor: 0,
            current_url: None,
            items: Vec::new(),
            cursor: 0,
            scroll_offset: 0,
            status_message: String::new(),
            downloading: false,
            download_progress: None,
            navigation_stack: Vec::new(),
            frame_count: 0,
            toast: None,
            ascii_mode: false,
            github_token: None,
            download_path: None,
            full_tree: None,
            folder_sizes: HashMap::new(),
            cwd: false,
            no_folder: false,
            is_searching: false,
            search_query: String::new(),
            selected_paths: HashSet::new(),
        }
    }

    pub fn show_toast(&mut self, message: String, type_: ToastType) {
        self.toast = Some(Toast::new(message, type_));
    }

    pub fn move_up(&mut self) {
        if self.cursor > 0 {
            self.cursor -= 1;
        }
        self.adjust_scroll();
    }

    pub fn move_down(&mut self, item_count: usize) {
        if self.cursor < item_count.saturating_sub(1) {
            self.cursor += 1;
        }
        self.adjust_scroll();
    }

    pub fn move_top(&mut self) {
        self.cursor = 0;
        self.adjust_scroll();
    }

    pub fn move_bottom(&mut self, item_count: usize) {
        if item_count > 0 {
            self.cursor = item_count - 1;
        }
        self.adjust_scroll();
    }

    fn adjust_scroll(&mut self) {
        if self.cursor < self.scroll_offset {
            self.scroll_offset = self.cursor;
        } else if self.cursor >= self.scroll_offset + LIST_VISIBLE_HEIGHT {
            self.scroll_offset = self.cursor - LIST_VISIBLE_HEIGHT + 1;
        }
    }

    pub fn get_view_items(&self) -> Vec<RepoItem> {
        let mut items = if self.is_searching {
            let source = self.full_tree.as_ref().unwrap_or(&self.items);
            let query = self.search_query.to_lowercase();
            let mut matches: Vec<_> = source
                .iter()
                .filter(|item| item.path.to_lowercase().contains(&query))
                .cloned()
                .collect();
            matches.sort_by(|a, b| a.path.cmp(&b.path));
            matches
        } else {
            self.items.clone()
        };

        for item in &mut items {
            item.selected = self.selected_paths.contains(&item.path);
        }
        items
    }

    pub fn sync_selections(&mut self) {
        for item in &mut self.items {
            item.selected = self.selected_paths.contains(&item.path);
        }
    }

    pub fn set_items(&mut self, items: Vec<RepoItem>) {
        self.items = items;
        self.sync_selections();
    }

    pub fn reset_browser_position(&mut self, cursor: usize) {
        self.cursor = cursor;
        self.scroll_offset = 0;
        self.adjust_scroll();
    }

    pub fn clear_search(&mut self) {
        self.is_searching = false;
        self.search_query.clear();
        self.reset_browser_position(0);
    }

    pub fn toggle_selection_at_cursor(&mut self) {
        let items = self.get_view_items();
        if let Some(item) = items.get(self.cursor) {
            if self.selected_paths.contains(&item.path) {
                self.selected_paths.remove(&item.path);
            } else {
                self.selected_paths.insert(item.path.clone());
            }
        }
    }

    pub fn select_visible(&mut self, selected: bool) {
        for item in self.get_view_items() {
            if selected {
                self.selected_paths.insert(item.path);
            } else {
                self.selected_paths.remove(&item.path);
            }
        }
    }

    pub fn get_selected_items(&self) -> Vec<RepoItem> {
        if let Some(full_tree) = &self.full_tree {
            full_tree
                .iter()
                .filter(|i| self.selected_paths.contains(&i.path))
                .cloned()
                .map(|mut i| {
                    i.selected = true;
                    i
                })
                .collect()
        } else {
            self.items
                .iter()
                .filter(|i| self.selected_paths.contains(&i.path))
                .cloned()
                .map(|mut i| {
                    i.selected = true;
                    i
                })
                .collect()
        }
    }
}

pub async fn run_tui(
    initial_url: Option<String>,
    token: Option<String>,
    download_path: Option<String>,
    cwd: bool,
    no_folder: bool,
) -> Result<()> {
    install_panic_hook();
    enable_raw_mode().context("Failed to enable raw mode")?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen).context("Failed to enter alternate screen")?;

    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend).context("Failed to create terminal")?;

    let client = GitHubClient::new(token.clone())?;
    let mut state_init = AppState::new();
    state_init.github_token = token;
    state_init.download_path = download_path;
    state_init.cwd = cwd;
    state_init.no_folder = no_folder;

    // Auto-detect local git remote if no URL provided
    if initial_url.is_none() {
        if let Some(remote) = GitHubUrl::get_local_git_remote() {
            state_init.url_input = remote;
            state_init.url_cursor = state_init.url_input.len();
        }
    }

    let has_initial_url = initial_url.is_some();
    if let Some(url) = initial_url {
        state_init.url_input = url;
        state_init.mode = AppMode::Searching;
        state_init.status_message = "Parsing URL...".to_string();
    }

    let state = Arc::new(Mutex::new(state_init));

    if has_initial_url {
        spawn_repo_load(state.clone(), client.clone());
    }

    let result = event_loop(&mut terminal, state, client).await;

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    result
}

async fn event_loop(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    state: Arc<Mutex<AppState>>,
    client: GitHubClient,
) -> Result<()> {
    loop {
        {
            let mut state_lock = state.lock().await;
            state_lock.frame_count = state_lock.frame_count.wrapping_add(1);
            let frame_count = state_lock.frame_count;

            if let Some(ref t) = state_lock.toast {
                if t.is_expired() {
                    state_lock.toast = None;
                }
            }

            terminal.draw(|f| {
                let size = f.area();
                f.render_widget(
                    ratatui::widgets::Block::default()
                        .style(ratatui::style::Style::default().bg(theme::BG_COLOR)),
                    size,
                );

                match state_lock.mode {
                    AppMode::Input => {
                        let cursor_visible = (frame_count / 5) % 2 == 0;
                        components::input::render(
                            f,
                            size,
                            &state_lock.url_input,
                            state_lock.url_cursor,
                            &state_lock.status_message,
                            cursor_visible,
                        );
                    }
                    AppMode::Searching => {
                        components::searching::render(
                            f,
                            size,
                            frame_count,
                            &state_lock.status_message,
                        );
                    }
                    AppMode::Browse => {
                        let filtered_items = state_lock.get_view_items();

                        let (dl_progress, dl_file) =
                            if let Some(ref progress) = state_lock.download_progress {
                                let completed = progress.completed.load(Ordering::Relaxed);
                                let file = progress
                                    .current_file
                                    .try_lock()
                                    .map(|f| f.clone())
                                    .unwrap_or_default();
                                (Some((completed, progress.total)), file)
                            } else {
                                (None, String::new())
                            };

                        let browser_state = components::browser::BrowserState {
                            items: &filtered_items,
                            current_url: state_lock.current_url.as_ref(),
                            cursor: state_lock.cursor,
                            scroll_offset: state_lock.scroll_offset,
                            is_downloading: state_lock.downloading,
                            download_progress: dl_progress,
                            download_current_file: &dl_file,
                            ascii_mode: state_lock.ascii_mode,
                            folder_sizes: &state_lock.folder_sizes,
                            is_searching: state_lock.is_searching,
                            search_query: &state_lock.search_query,
                            selected_count: state_lock.selected_paths.len(),
                        };
                        components::browser::render(f, size, &browser_state);
                    }
                }

                if let Some(ref toast) = state_lock.toast {
                    components::toast::render(f, size, toast);
                }
            })?;
        }

        if event::poll(std::time::Duration::from_millis(80))? {
            if let Event::Key(key) = event::read()? {
                if key.kind == event::KeyEventKind::Press
                    && handle_input(key, state.clone(), &client).await?
                {
                    break;
                }
            }
        }
    }

    Ok(())
}

async fn handle_input(
    key: KeyEvent,
    state: Arc<Mutex<AppState>>,
    client: &GitHubClient,
) -> Result<bool> {
    let state_handle = state.clone();
    let mut app_state = state.lock().await;
    if (key.code == KeyCode::Char('q') || key.code == KeyCode::Char('c'))
        && key.modifiers.contains(KeyModifiers::CONTROL)
    {
        return Ok(true);
    }

    match app_state.mode {
        AppMode::Input => Ok(handle_input_mode_input(key, &mut app_state, &state, client)),
        AppMode::Searching => {
            handle_input_mode_searching(key, &mut app_state);
            Ok(false)
        }
        AppMode::Browse => handle_input_mode_browse(key, app_state, state_handle, client).await,
    }
}

fn handle_input_mode_input(
    key: KeyEvent,
    app_state: &mut AppState,
    state: &Arc<Mutex<AppState>>,
    client: &GitHubClient,
) -> bool {
    match key.code {
        KeyCode::Char('w' | 'u') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            app_state.url_input.clear();
            app_state.url_cursor = 0;
        }
        KeyCode::Char(c)
            if !key
                .modifiers
                .intersects(KeyModifiers::CONTROL | KeyModifiers::ALT | KeyModifiers::SUPER) =>
        {
            let pos = app_state.url_cursor;
            app_state.url_input.insert(pos, c);
            app_state.url_cursor += 1;
        }
        KeyCode::Backspace => {
            if key.modifiers.contains(KeyModifiers::CONTROL)
                || key.modifiers.contains(KeyModifiers::ALT)
                || key.modifiers.contains(KeyModifiers::SUPER)
            {
                app_state.url_input.clear();
                app_state.url_cursor = 0;
            } else if app_state.url_cursor > 0 {
                let pos = app_state.url_cursor;
                app_state.url_input.remove(pos - 1);
                app_state.url_cursor -= 1;
            }
        }
        KeyCode::Delete => {
            app_state.url_input.clear();
            app_state.url_cursor = 0;
        }
        KeyCode::Left => {
            if app_state.url_cursor > 0 {
                app_state.url_cursor -= 1;
            }
        }
        KeyCode::Right => {
            if app_state.url_cursor < app_state.url_input.len() {
                app_state.url_cursor += 1;
            }
        }
        KeyCode::Home => {
            app_state.url_cursor = 0;
        }
        KeyCode::End => {
            app_state.url_cursor = app_state.url_input.len();
        }
        KeyCode::Tab => {
            let target = "https://github.com/";
            if app_state.url_input.is_empty()
                || (target.starts_with(&app_state.url_input)
                    && app_state.url_input.len() < target.len())
            {
                app_state.url_input = target.to_string();
                app_state.url_cursor = app_state.url_input.len();
            }
        }
        KeyCode::Esc => return true,
        KeyCode::Enter => {
            if app_state.url_input.trim().is_empty() {
                app_state.show_toast("Please enter a GitHub URL".to_string(), ToastType::Warning);
            } else {
                app_state.mode = AppMode::Searching;
                app_state.status_message = "Parsing URL...".to_string();
                spawn_repo_load(state.clone(), client.clone());
            }
        }
        _ => {}
    }

    false
}

fn handle_input_mode_searching(key: KeyEvent, app_state: &mut AppState) {
    if key.code == KeyCode::Esc {
        app_state.mode = AppMode::Input;
        app_state.status_message.clear();
    }
}

async fn handle_input_mode_browse(
    key: KeyEvent,
    mut app_state: tokio::sync::MutexGuard<'_, AppState>,
    state: Arc<Mutex<AppState>>,
    client: &GitHubClient,
) -> Result<bool> {
    match key.code {
        KeyCode::Char('q' | 'Q') if !app_state.is_searching => return Ok(true),
        KeyCode::Esc if !app_state.is_searching => {
            app_state.mode = AppMode::Input;
            app_state.selected_paths.clear();
            return Ok(false);
        }
        KeyCode::Esc if app_state.is_searching => {
            app_state.clear_search();
        }
        KeyCode::Enter if app_state.is_searching => {
            app_state.is_searching = false;
        }
        KeyCode::Char('i') if !app_state.is_searching => toggle_ascii_mode(&mut app_state),
        KeyCode::Up | KeyCode::Char('k') if !app_state.is_searching => app_state.move_up(),
        KeyCode::Down | KeyCode::Char('j') if !app_state.is_searching => {
            let count = app_state.get_view_items().len();
            app_state.move_down(count);
        }
        KeyCode::Up if app_state.is_searching => app_state.move_up(),
        KeyCode::Down if app_state.is_searching => {
            let count = app_state.get_view_items().len();
            app_state.move_down(count);
        }
        KeyCode::Home | KeyCode::Char('g') if !app_state.is_searching => app_state.move_top(),
        KeyCode::End | KeyCode::Char('G') if !app_state.is_searching => {
            let count = app_state.get_view_items().len();
            app_state.move_bottom(count);
        }
        KeyCode::PageUp => move_page_up(&mut app_state),
        KeyCode::PageDown => move_page_down(&mut app_state),
        KeyCode::Char(' ') => app_state.toggle_selection_at_cursor(),
        KeyCode::Char('a') if !app_state.is_searching => app_state.select_visible(true),
        KeyCode::Char('u') if !app_state.is_searching => app_state.select_visible(false),
        KeyCode::Char('/') if !app_state.is_searching => begin_search(&mut app_state),
        KeyCode::Char(c) if app_state.is_searching => {
            app_state.search_query.push(c);
            app_state.reset_browser_position(0);
        }
        KeyCode::Backspace if app_state.is_searching => {
            app_state.search_query.pop();
            app_state.reset_browser_position(0);
        }
        KeyCode::Backspace | KeyCode::Left | KeyCode::Char('h') if !app_state.is_searching => {
            let navigation = prepare_back_navigation(&mut app_state);
            drop(app_state);
            apply_back_navigation(navigation, state, client).await?;
        }
        KeyCode::Enter | KeyCode::Right | KeyCode::Char('l') if !app_state.is_searching => {
            let navigation = prepare_enter_directory(&mut app_state);
            drop(app_state);
            apply_enter_directory(navigation, state, client).await?;
        }
        KeyCode::Char('d' | 'D') if !app_state.is_searching => {
            maybe_start_download(&mut app_state, state);
        }
        _ => {}
    }

    Ok(false)
}

fn toggle_ascii_mode(app_state: &mut AppState) {
    app_state.ascii_mode = !app_state.ascii_mode;
    let mode_name = if app_state.ascii_mode {
        "ASCII mode"
    } else {
        "Icon mode"
    };
    app_state.show_toast(mode_name.to_string(), ToastType::Info);
}

fn move_page_up(app_state: &mut AppState) {
    for _ in 0..PAGE_STEP {
        app_state.move_up();
    }
}

fn move_page_down(app_state: &mut AppState) {
    let count = app_state.get_view_items().len();
    for _ in 0..PAGE_STEP {
        app_state.move_down(count);
    }
}

fn begin_search(app_state: &mut AppState) {
    app_state.is_searching = true;
    app_state.search_query.clear();
    app_state.reset_browser_position(0);
}

fn prepare_back_navigation(app_state: &mut AppState) -> BackNavigation {
    if let Some((prev_url, prev_cursor)) = app_state.navigation_stack.pop() {
        if let Some(full_tree) = &app_state.full_tree {
            let items = repo_items_for_path(full_tree, &prev_url.path);
            let cursor = clamp_cursor(prev_cursor, items.len());
            BackNavigation::Local {
                items,
                url: prev_url,
                cursor,
            }
        } else {
            BackNavigation::Remote {
                url: prev_url,
                cursor: prev_cursor,
            }
        }
    } else {
        BackNavigation::ExitToInput
    }
}

async fn apply_back_navigation(
    navigation: BackNavigation,
    state: Arc<Mutex<AppState>>,
    client: &GitHubClient,
) -> Result<()> {
    match navigation {
        BackNavigation::ExitToInput => {
            let mut app_state = state.lock().await;
            app_state.mode = AppMode::Input;
        }
        BackNavigation::Local { items, url, cursor } => {
            let mut app_state = state.lock().await;
            app_state.set_items(items);
            app_state.current_url = Some(url);
            app_state.reset_browser_position(cursor);
        }
        BackNavigation::Remote { url, cursor } => match client.fetch_contents(&url.api_url()).await
        {
            Ok(mut items) => {
                sort_repo_items(&mut items);
                let cursor = clamp_cursor(cursor, items.len());
                let mut app_state = state.lock().await;
                app_state.set_items(items);
                app_state.current_url = Some(url);
                app_state.reset_browser_position(cursor);
            }
            Err(error) => {
                let mut app_state = state.lock().await;
                app_state.show_toast(format!("Nav Error: {error}"), ToastType::Error);
            }
        },
    }

    Ok(())
}

fn prepare_enter_directory(app_state: &mut AppState) -> EnterDirectory {
    let items = app_state.get_view_items();
    let Some(item) = items.get(app_state.cursor).cloned() else {
        return EnterDirectory::None;
    };
    if !item.is_dir() {
        return EnterDirectory::None;
    }

    let cursor_pos = app_state.cursor;
    let Some(current_url) = app_state.current_url.clone() else {
        return EnterDirectory::None;
    };

    app_state
        .navigation_stack
        .push((current_url.clone(), cursor_pos));

    let next_url = GitHubUrl {
        path: item.path,
        ..current_url
    };

    if let Some(full_tree) = &app_state.full_tree {
        EnterDirectory::Local {
            items: repo_items_for_path(full_tree, &next_url.path),
            url: next_url,
        }
    } else {
        EnterDirectory::Remote { url: next_url }
    }
}

async fn apply_enter_directory(
    navigation: EnterDirectory,
    state: Arc<Mutex<AppState>>,
    client: &GitHubClient,
) -> Result<()> {
    match navigation {
        EnterDirectory::None => {}
        EnterDirectory::Local { items, url } => {
            let mut app_state = state.lock().await;
            app_state.set_items(items);
            app_state.current_url = Some(url);
            app_state.reset_browser_position(0);
        }
        EnterDirectory::Remote { url } => match client.fetch_contents(&url.api_url()).await {
            Ok(mut items) => {
                sort_repo_items(&mut items);
                let mut app_state = state.lock().await;
                app_state.set_items(items);
                app_state.current_url = Some(url);
                app_state.reset_browser_position(0);
            }
            Err(error) => {
                let mut app_state = state.lock().await;
                app_state.navigation_stack.pop();
                app_state.show_toast(format!("Nav Error: {error}"), ToastType::Error);
            }
        },
    }

    Ok(())
}

fn maybe_start_download(app_state: &mut AppState, state: Arc<Mutex<AppState>>) {
    if app_state.get_selected_items().is_empty() {
        app_state.show_toast(
            "No items selected! Use Space to select.".to_string(),
            ToastType::Warning,
        );
        return;
    }

    if app_state.downloading {
        return;
    }

    app_state.downloading = true;
    tokio::spawn(async move {
        let _ = perform_download(state).await;
    });
}

fn spawn_repo_load(state: Arc<Mutex<AppState>>, client: GitHubClient) {
    tokio::spawn(async move {
        let url = {
            let s = state.lock().await;
            s.url_input.clone()
        };

        let state_for_load = state.clone();
        let client_for_load = client.clone();
        match GitHubUrl::parse(&url) {
            Ok(gh_url) => load_repo(state_for_load, client_for_load, gh_url).await,
            Err(e) => {
                let mut s = state.lock().await;
                s.mode = AppMode::Input;
                s.show_toast(format!("Invalid URL: {e}"), ToastType::Error);
            }
        }
    });
}

async fn load_repo(state: Arc<Mutex<AppState>>, client: GitHubClient, mut gh_url: GitHubUrl) {
    let state_c = state.clone();
    let mut current_client = client;

    {
        let mut s = state_c.lock().await;
        s.status_message = "Fetching repository tree...".to_string();
        s.mode = AppMode::Searching;
    }

    let mut tree_result = current_client
        .fetch_recursive_tree(&gh_url.owner, &gh_url.repo, &gh_url.branch)
        .await;

    // Handle invalid token - fallback to public API
    if let Err(GitHubError::InvalidToken) = &tree_result {
        {
            let mut s = state_c.lock().await;
            s.show_toast(
                "Invalid token! Falling back to public API.".to_string(),
                ToastType::Warning,
            );
        }
        if let Ok(no_auth_client) = GitHubClient::new(None) {
            current_client = no_auth_client;
            tree_result = current_client
                .fetch_recursive_tree(&gh_url.owner, &gh_url.repo, &gh_url.branch)
                .await;
        }
    }

    // Try master branch if main fails
    if let Err(GitHubError::NotFound(_)) = &tree_result {
        if gh_url.branch == "main" {
            gh_url.branch = "master".to_string();
            {
                let mut s = state_c.lock().await;
                s.status_message = "Trying master branch...".to_string();
            }
            tree_result = current_client
                .fetch_recursive_tree(&gh_url.owner, &gh_url.repo, &gh_url.branch)
                .await;
        }
    }

    match tree_result {
        Ok(tree_response) if !tree_response.truncated => {
            let items =
                map_tree_to_items(tree_response, &gh_url.owner, &gh_url.repo, &gh_url.branch);
            let folder_sizes = calculate_folder_sizes(&items);
            let (browse_path, cursor_path) = resolve_requested_view(&items, &gh_url.path);
            let mut current_items = repo_items_for_path(&items, &browse_path);

            current_client
                .resolve_lfs_files(
                    &mut current_items,
                    &gh_url.owner,
                    &gh_url.repo,
                    &gh_url.branch,
                )
                .await;

            let cursor = cursor_path
                .as_deref()
                .and_then(|path| find_cursor_by_path(&current_items, path))
                .unwrap_or(0);

            let mut s = state_c.lock().await;
            s.full_tree = Some(items);
            s.folder_sizes = folder_sizes;
            s.set_items(current_items);
            s.current_url = Some(GitHubUrl {
                path: browse_path,
                ..gh_url
            });
            s.reset_browser_position(cursor);
            s.mode = AppMode::Browse;
            s.status_message = String::new();
            s.show_toast("Repository loaded!".to_string(), ToastType::Success);
        }
        Ok(_) | Err(_) => {
            {
                let mut s = state_c.lock().await;
                s.status_message =
                    "Tree too large, falling back to folder-by-folder mode...".to_string();
                s.full_tree = None;
                s.folder_sizes.clear();
            }

            let requested_path = gh_url.path.clone();
            let selected_file_path = (!requested_path.is_empty()).then_some(requested_path.clone());
            let initial_result = current_client.fetch_contents(&gh_url.api_url()).await;

            let (browse_url, result) = match initial_result {
                Ok(items) if is_exact_file_match(&items, &requested_path) => {
                    let mut browse_url = gh_url.clone();
                    browse_url.path = parent_repo_path(&requested_path).unwrap_or_default();
                    let result = current_client.fetch_contents(&browse_url.api_url()).await;
                    (browse_url, result)
                }
                other => (gh_url.clone(), other),
            };

            match result {
                Err(e) => {
                    let mut s = state_c.lock().await;
                    s.mode = AppMode::Input;
                    let err_msg = if let Some(gh_err) = e.downcast_ref::<GitHubError>() {
                        match gh_err {
                            GitHubError::RateLimitReached(user) => {
                                format!("Rate limit reached for {}. Add a token for more!", user)
                            }
                            GitHubError::NotFound(_) => "Repository or path not found.".to_string(),
                            _ => format!("Error: {}", gh_err),
                        }
                    } else {
                        format!("Error: {}", e)
                    };
                    s.show_toast(err_msg, ToastType::Error);
                }
                Ok(mut items) => {
                    sort_repo_items(&mut items);

                    current_client
                        .resolve_lfs_files(&mut items, &gh_url.owner, &gh_url.repo, &gh_url.branch)
                        .await;

                    let cursor = selected_file_path
                        .as_deref()
                        .and_then(|path| find_cursor_by_path(&items, path))
                        .unwrap_or(0);

                    let mut s = state_c.lock().await;
                    s.set_items(items);
                    s.current_url = Some(browse_url);
                    s.reset_browser_position(cursor);
                    s.mode = AppMode::Browse;
                    s.status_message = String::new();
                    s.show_toast("Repository loaded!".to_string(), ToastType::Success);
                }
            }
        }
    }
}

async fn perform_download(state: Arc<Mutex<AppState>>) -> Result<()> {
    use crate::download::Downloader;

    let (selected_items, current_url, full_tree, token, custom_path, cwd, no_folder) = {
        let s = state.lock().await;
        if let Some(url) = &s.current_url {
            (
                s.get_selected_items(),
                url.clone(),
                s.full_tree.clone(),
                s.github_token.clone(),
                s.download_path.clone(),
                s.cwd,
                s.no_folder,
            )
        } else {
            return Ok(());
        }
    };

    let outcome: Result<DownloadOutcome> = async {
        let download_client = GitHubClient::new(token.clone())?;
        let mut items_to_download =
            collect_download_items(&download_client, &selected_items, full_tree.as_deref()).await?;
        dedupe_repo_items(&mut items_to_download);
        download_client
            .resolve_lfs_files(
                &mut items_to_download,
                &current_url.owner,
                &current_url.repo,
                &current_url.branch,
            )
            .await;

        if items_to_download.is_empty() {
            return Ok(DownloadOutcome::Empty);
        }

        let download_dir = if cwd {
            std::env::current_dir().context("Could not get current working directory")?
        } else if let Some(path) = custom_path {
            std::path::PathBuf::from(path)
        } else {
            dirs::download_dir()
                .or_else(|| dirs::home_dir().map(|h| h.join("Downloads")))
                .context("Could not find Downloads directory")?
        };

        let download_dir = if no_folder {
            download_dir
        } else {
            download_dir.join(&current_url.repo)
        };

        let progress = Arc::new(DownloadProgress {
            total: items_to_download.len(),
            completed: std::sync::atomic::AtomicUsize::new(0),
            current_file: tokio::sync::Mutex::new(String::new()),
        });

        {
            let mut s = state.lock().await;
            s.download_progress = Some(progress.clone());
        }

        let downloader = Downloader::new(download_dir.clone(), token)?;
        let errors = downloader
            .download_items(&items_to_download, progress)
            .await?;

        Ok(DownloadOutcome::Completed {
            download_dir,
            errors,
        })
    }
    .await;

    let mut s = state.lock().await;
    s.downloading = false;
    s.download_progress = None;

    match outcome {
        Ok(DownloadOutcome::Empty) => {
            s.show_toast(
                "No downloadable files found in the selection.".to_string(),
                ToastType::Warning,
            );
        }
        Ok(DownloadOutcome::Completed {
            download_dir,
            errors,
        }) => {
            if errors.is_empty() {
                s.show_toast(
                    format!("Downloaded to: {}", download_dir.display()),
                    ToastType::Success,
                );
            } else {
                s.show_toast(
                    format!("Done with {} errors", errors.len()),
                    ToastType::Warning,
                );
            }
        }
        Err(e) => {
            s.show_toast(format!("Download failed: {}", e), ToastType::Error);
        }
    }

    Ok(())
}

fn calculate_folder_sizes(items: &[RepoItem]) -> HashMap<String, u64> {
    let mut sizes = HashMap::new();
    for item in items {
        if item.is_file() {
            let path = &item.path;
            let parts: Vec<&str> = path.split('/').collect();
            for i in 1..parts.len() {
                let parent_path = parts[..i].join("/");
                if !parent_path.is_empty() {
                    let entry = sizes.entry(parent_path).or_insert(0);
                    *entry += item.actual_size().unwrap_or(0);
                }
            }
        }
    }
    sizes
}

fn sort_repo_items(items: &mut [RepoItem]) {
    items.sort_by(|a, b| {
        let a_dir = a.is_dir();
        let b_dir = b.is_dir();
        b_dir
            .cmp(&a_dir)
            .then(a.name.to_lowercase().cmp(&b.name.to_lowercase()))
    });
}

fn repo_items_for_path(items: &[RepoItem], current_path: &str) -> Vec<RepoItem> {
    let mut visible_items: Vec<RepoItem> = if current_path.is_empty() {
        items
            .iter()
            .filter(|item| !item.path.contains('/'))
            .cloned()
            .collect()
    } else {
        let prefix = format!("{current_path}/");
        items
            .iter()
            .filter(|item| {
                item.path.starts_with(&prefix) && !item.path[prefix.len()..].contains('/')
            })
            .cloned()
            .collect()
    };

    sort_repo_items(&mut visible_items);
    visible_items
}

fn resolve_requested_view(items: &[RepoItem], requested_path: &str) -> (String, Option<String>) {
    if requested_path.is_empty() {
        return (String::new(), None);
    }

    if items
        .iter()
        .any(|item| item.path == requested_path && item.is_file())
    {
        (
            parent_repo_path(requested_path).unwrap_or_default(),
            Some(requested_path.to_string()),
        )
    } else {
        (requested_path.to_string(), None)
    }
}

fn parent_repo_path(path: &str) -> Option<String> {
    path.rsplit_once('/').map(|(parent, _)| parent.to_string())
}

fn find_cursor_by_path(items: &[RepoItem], path: &str) -> Option<usize> {
    items.iter().position(|item| item.path == path)
}

fn clamp_cursor(cursor: usize, item_count: usize) -> usize {
    item_count.saturating_sub(1).min(cursor)
}

fn is_exact_file_match(items: &[RepoItem], requested_path: &str) -> bool {
    items.len() == 1 && items[0].is_file() && items[0].path == requested_path
}

async fn collect_download_items(
    client: &GitHubClient,
    selected_items: &[RepoItem],
    full_tree: Option<&[RepoItem]>,
) -> Result<Vec<RepoItem>> {
    let mut download_items = Vec::new();

    for item in selected_items {
        if item.is_file() {
            download_items.push(downloadable_item(item.clone()));
            continue;
        }

        if let Some(full_tree) = full_tree {
            let prefix = format!("{}/", item.path);
            download_items.extend(
                full_tree
                    .iter()
                    .filter(|tree_item| tree_item.is_file() && tree_item.path.starts_with(&prefix))
                    .cloned()
                    .map(downloadable_item),
            );
        } else {
            download_items.extend(collect_directory_files(client, item).await?);
        }
    }

    Ok(download_items)
}

async fn collect_directory_files(client: &GitHubClient, root: &RepoItem) -> Result<Vec<RepoItem>> {
    let mut pending = vec![root.clone()];
    let mut files = Vec::new();

    while let Some(directory) = pending.pop() {
        for item in client.fetch_contents(&directory.url).await? {
            if item.is_file() {
                files.push(downloadable_item(item));
            } else {
                pending.push(item);
            }
        }
    }

    Ok(files)
}

fn downloadable_item(mut item: RepoItem) -> RepoItem {
    item.name = item.path.clone();
    item.selected = true;
    item
}

fn dedupe_repo_items(items: &mut Vec<RepoItem>) {
    let mut seen_paths = HashSet::new();
    items.retain(|item| seen_paths.insert(item.path.clone()));
    items.sort_by(|a, b| a.path.cmp(&b.path));
}

fn map_tree_to_items(
    tree: crate::github::GitTreeResponse,
    owner: &str,
    repo: &str,
    branch: &str,
) -> Vec<RepoItem> {
    tree.tree
        .into_iter()
        .map(|entry| {
            let name = entry
                .path
                .split('/')
                .next_back()
                .unwrap_or(&entry.path)
                .to_string();
            let item_type = if entry.entry_type == "tree" {
                "dir".to_string()
            } else {
                "file".to_string()
            };

            let download_url = if item_type == "file" {
                Some(format!(
                    "https://raw.githubusercontent.com/{}/{}/{}/{}",
                    owner, repo, branch, entry.path
                ))
            } else {
                None
            };

            RepoItem {
                name,
                item_type,
                url: format!(
                    "https://api.github.com/repos/{}/{}/contents/{}?ref={}",
                    owner, repo, &entry.path, branch
                ),
                path: entry.path,
                download_url,
                size: entry.size,
                selected: false,
                lfs_oid: None,
                lfs_size: None,
                lfs_download_url: None,
            }
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn repo_item(path: &str, item_type: &str) -> RepoItem {
        RepoItem {
            name: path.rsplit('/').next().unwrap_or(path).to_string(),
            item_type: item_type.to_string(),
            path: path.to_string(),
            download_url: Some(format!("https://example.com/{path}")),
            url: format!("https://api.example.com/{path}"),
            size: Some(10),
            selected: false,
            lfs_oid: None,
            lfs_size: None,
            lfs_download_url: None,
        }
    }

    #[test]
    fn repo_items_for_path_returns_direct_children_sorted() {
        let items = vec![
            repo_item("src/lib.rs", "file"),
            repo_item("src/ui", "dir"),
            repo_item("src/main.rs", "file"),
            repo_item("README.md", "file"),
        ];

        let root_items = repo_items_for_path(&items, "");
        assert_eq!(root_items.len(), 1);
        assert_eq!(root_items[0].path, "README.md");

        let src_items = repo_items_for_path(&items, "src");
        assert_eq!(
            src_items
                .iter()
                .map(|item| item.path.as_str())
                .collect::<Vec<_>>(),
            vec!["src/ui", "src/lib.rs", "src/main.rs",]
        );
    }

    #[test]
    fn resolve_requested_view_targets_parent_for_files() {
        let items = vec![repo_item("src/lib.rs", "file"), repo_item("src", "dir")];
        let (browse_path, selected_path) = resolve_requested_view(&items, "src/lib.rs");

        assert_eq!(browse_path, "src");
        assert_eq!(selected_path.as_deref(), Some("src/lib.rs"));
    }

    #[test]
    fn dedupe_repo_items_removes_duplicate_paths() {
        let mut items = vec![
            downloadable_item(repo_item("src/lib.rs", "file")),
            downloadable_item(repo_item("src/lib.rs", "file")),
            downloadable_item(repo_item("src/main.rs", "file")),
        ];

        dedupe_repo_items(&mut items);

        assert_eq!(items.len(), 2);
        assert_eq!(items[0].path, "src/lib.rs");
        assert_eq!(items[1].path, "src/main.rs");
    }

    #[test]
    fn calculate_folder_sizes_accumulates_nested_file_sizes() {
        let mut root_file = repo_item("src/main.rs", "file");
        root_file.size = Some(20);
        let mut nested_file = repo_item("src/ui/mod.rs", "file");
        nested_file.size = Some(30);

        let sizes = calculate_folder_sizes(&[root_file, nested_file]);

        assert_eq!(sizes.get("src"), Some(&50));
        assert_eq!(sizes.get("src/ui"), Some(&30));
    }
}
