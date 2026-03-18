use ratatui::{
    Frame,
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Modifier, Style},
    text::{Line, Span},
    widgets::{
        Block, Borders, List, ListItem, Paragraph, Scrollbar, ScrollbarOrientation, ScrollbarState,
    },
};

use crate::github::{GitHubUrl, RepoItem};
use crate::ui::theme::{
    ACCENT_COLOR, BG_COLOR, BORDER_COLOR, ERROR_COLOR, FG_COLOR, FOLDER_COLOR, HIGHLIGHT_BG,
    MAUVE_COLOR, PEACH_COLOR, SUBTEXT_COLOR, SUCCESS_COLOR, SURFACE_COLOR, WARNING_COLOR,
};
use std::collections::HashMap;

pub struct BrowserState<'a> {
    pub items: &'a [RepoItem],
    pub current_url: Option<&'a GitHubUrl>,
    pub cursor: usize,
    pub scroll_offset: usize,
    pub is_downloading: bool,
    pub download_progress: Option<(usize, usize)>,
    pub download_current_file: &'a str,
    pub ascii_mode: bool,
    pub folder_sizes: &'a HashMap<String, u64>,
    pub is_searching: bool,
    pub search_query: &'a str,
    pub selected_count: usize,
}

pub fn render(f: &mut Frame, area: Rect, state: &BrowserState) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3),
            Constraint::Min(10),
            if state.is_downloading {
                Constraint::Length(3)
            } else {
                Constraint::Length(0)
            },
            if state.is_searching {
                Constraint::Length(3)
            } else {
                Constraint::Length(0)
            },
            Constraint::Length(2),
        ])
        .split(area);

    render_header(f, chunks[0], state);
    render_file_list(f, chunks[1], state);
    if state.is_downloading {
        render_download_progress(f, chunks[2], state);
    }
    if state.is_searching {
        render_search_bar(f, chunks[3], state);
    }
    render_help_bar(f, chunks[4]);
}

fn render_header(f: &mut Frame, area: Rect, state: &BrowserState) {
    let breadcrumb_text = repository_breadcrumb(state.current_url);
    let selected_info = if state.selected_count > 0 {
        format!("  [{} selected]", state.selected_count)
    } else {
        String::new()
    };

    let header = Paragraph::new(Line::from(vec![
        Span::styled(
            breadcrumb_text,
            Style::default().fg(FG_COLOR).add_modifier(Modifier::BOLD),
        ),
        Span::styled(selected_info, Style::default().fg(SUCCESS_COLOR)),
    ]))
    .block(
        Block::default()
            .borders(Borders::ALL)
            .title(Span::styled(
                " Repository ",
                Style::default()
                    .fg(ACCENT_COLOR)
                    .add_modifier(Modifier::BOLD),
            ))
            .border_style(Style::default().fg(ACCENT_COLOR))
            .style(Style::default().bg(BG_COLOR)),
    );
    f.render_widget(header, area);
}

fn render_file_list(f: &mut Frame, area: Rect, state: &BrowserState) {
    let mut rows = vec![build_list_header()];
    rows.extend(
        state
            .items
            .iter()
            .enumerate()
            .skip(state.scroll_offset)
            .map(|(index, item)| build_file_row(index, item, state)),
    );

    let list = List::new(rows).block(
        Block::default()
            .borders(Borders::ALL)
            .title(format!(" Files ({}) ", state.items.len()))
            .border_style(Style::default().fg(if state.is_downloading {
                WARNING_COLOR
            } else {
                BORDER_COLOR
            }))
            .style(Style::default().bg(BG_COLOR)),
    );
    f.render_widget(list, area);

    if state.items.len() > 10 {
        let mut scrollbar_state =
            ScrollbarState::new(state.items.len()).position(state.scroll_offset);
        f.render_stateful_widget(
            Scrollbar::new(ScrollbarOrientation::VerticalRight)
                .begin_symbol(Some("^"))
                .end_symbol(Some("v"))
                .track_symbol(Some("|"))
                .thumb_symbol("█"),
            area,
            &mut scrollbar_state,
        );
    }
}

fn build_list_header() -> ListItem<'static> {
    let header_line = Line::from(vec![
        Span::styled("    ", Style::default().bg(SURFACE_COLOR)),
        Span::styled(
            format!("{:<40}", "Name"),
            Style::default()
                .fg(FG_COLOR)
                .add_modifier(Modifier::BOLD)
                .bg(SURFACE_COLOR),
        ),
        Span::styled("  ", Style::default().bg(SURFACE_COLOR)),
        Span::styled(
            format!("{:<7}", "Type"),
            Style::default()
                .fg(FG_COLOR)
                .add_modifier(Modifier::BOLD)
                .bg(SURFACE_COLOR),
        ),
        Span::styled("  ", Style::default().bg(SURFACE_COLOR)),
        Span::styled(
            format!("{:>10}", "Size"),
            Style::default()
                .fg(FG_COLOR)
                .add_modifier(Modifier::BOLD)
                .bg(SURFACE_COLOR),
        ),
        Span::styled("  ", Style::default().bg(SURFACE_COLOR)),
    ]);

    ListItem::new(header_line).style(Style::default().bg(SURFACE_COLOR))
}

fn build_file_row(index: usize, item: &RepoItem, state: &BrowserState) -> ListItem<'static> {
    let is_selected = index == state.cursor;
    let file_type = get_file_type(&item.name, item.is_dir());
    let display_name = if state.is_searching {
        &item.path
    } else {
        &item.name
    };
    let name_with_icon = format!(
        "{}{}",
        item_icon(item, state.ascii_mode),
        truncate_name(display_name, 34)
    );
    let name_display = format!("{name_with_icon:<40}");
    let content = Line::from(vec![
        selection_mark(item.selected),
        Span::styled(name_display, item_name_style(item, is_selected)),
        Span::styled("  ", Style::default()),
        Span::styled(
            format!("{file_type:<7}"),
            Style::default().fg(file_type_color(&file_type)),
        ),
        Span::styled("  ", Style::default()),
        Span::styled(
            size_display(item, state.folder_sizes),
            Style::default().fg(BORDER_COLOR),
        ),
    ]);

    let row = ListItem::new(content);
    if is_selected {
        row.style(Style::default().bg(HIGHLIGHT_BG))
    } else {
        row
    }
}

fn render_download_progress(f: &mut Frame, area: Rect, state: &BrowserState) {
    let (completed, total) = state.download_progress.unwrap_or((0, 0));
    let pct = progress_percentage(completed, total);
    let bar_width = usize::from(area.width).saturating_sub(30);
    let filled = bar_width.saturating_mul(pct).saturating_div(100);
    let empty = bar_width.saturating_sub(filled);
    let bar = format!("{}{}", "#".repeat(filled), "-".repeat(empty));
    let file_name = if state.download_current_file.is_empty() {
        "Starting..."
    } else {
        state.download_current_file
    };

    let progress = Paragraph::new(vec![
        Line::from(vec![
            Span::styled(
                " >> ",
                Style::default()
                    .fg(BG_COLOR)
                    .bg(SUCCESS_COLOR)
                    .add_modifier(Modifier::BOLD),
            ),
            Span::styled(
                format!(" [{bar}] {pct:>3}%  {completed}/{total}"),
                Style::default().fg(SUCCESS_COLOR),
            ),
        ]),
        Line::from(vec![
            Span::styled("    ", Style::default()),
            Span::styled(
                truncate_name(file_name, 60),
                Style::default().fg(SUBTEXT_COLOR),
            ),
        ]),
    ])
    .style(Style::default().bg(BG_COLOR));
    f.render_widget(progress, area);
}

fn render_search_bar(f: &mut Frame, area: Rect, state: &BrowserState) {
    let search_bar = Paragraph::new(Line::from(vec![
        Span::styled(
            " /",
            Style::default()
                .fg(MAUVE_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            format!(" {}", state.search_query),
            Style::default().fg(FG_COLOR),
        ),
        Span::styled("_", Style::default().fg(ACCENT_COLOR)),
    ]))
    .block(
        Block::default()
            .borders(Borders::ALL)
            .title(Span::styled(
                " Search (across all files) ",
                Style::default()
                    .fg(MAUVE_COLOR)
                    .add_modifier(Modifier::BOLD),
            ))
            .border_style(Style::default().fg(MAUVE_COLOR))
            .style(Style::default().bg(BG_COLOR)),
    );
    f.render_widget(search_bar, area);
}

fn render_help_bar(f: &mut Frame, area: Rect) {
    let help = Paragraph::new(Line::from(vec![
        Span::styled("  ", Style::default()),
        help_key("j/k", ACCENT_COLOR),
        Span::styled(" Nav", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("Enter", SUCCESS_COLOR),
        Span::styled(" Open/Preview", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("Space", WARNING_COLOR),
        Span::styled(" Select", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("a", FOLDER_COLOR),
        Span::styled("/", Style::default().fg(BORDER_COLOR)),
        help_key("u", FOLDER_COLOR),
        Span::styled(" All/None", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("d", SUCCESS_COLOR),
        Span::styled(" Download", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("/", MAUVE_COLOR),
        Span::styled(" Search", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("<-", ERROR_COLOR),
        Span::styled(" Back", Style::default().fg(BORDER_COLOR)),
        help_sep(),
        help_key("q", ERROR_COLOR),
        Span::styled(" Quit", Style::default().fg(BORDER_COLOR)),
    ]))
    .alignment(Alignment::Center)
    .style(Style::default().bg(BG_COLOR));
    f.render_widget(help, area);
}

fn repository_breadcrumb(current_url: Option<&GitHubUrl>) -> String {
    if let Some(url) = current_url {
        let path_display = if url.path.is_empty() {
            "/".to_string()
        } else {
            format!("/{}", url.path)
        };
        format!(
            " {}/{} @ {}  {}",
            url.owner, url.repo, url.branch, path_display
        )
    } else {
        " Loading...".to_string()
    }
}

fn selection_mark(selected: bool) -> Span<'static> {
    if selected {
        Span::styled(
            "[*] ",
            Style::default()
                .fg(SUCCESS_COLOR)
                .add_modifier(Modifier::BOLD),
        )
    } else {
        Span::styled("[ ] ", Style::default().fg(BORDER_COLOR))
    }
}

fn item_name_style(item: &RepoItem, is_selected: bool) -> Style {
    if is_selected {
        Style::default()
            .fg(ACCENT_COLOR)
            .add_modifier(Modifier::BOLD)
            .bg(HIGHLIGHT_BG)
    } else if item.is_dir() {
        Style::default().fg(FOLDER_COLOR)
    } else if item.is_lfs() {
        Style::default().fg(PEACH_COLOR)
    } else {
        Style::default().fg(FG_COLOR)
    }
}

fn file_type_color(file_type: &str) -> ratatui::style::Color {
    match file_type {
        "DIR" => FOLDER_COLOR,
        "RS" | "PY" | "JS" | "TS" | "GO" | "RB" | "C" | "CPP" | "JAVA" => MAUVE_COLOR,
        "MD" | "TXT" | "JSON" | "YAML" | "TOML" => WARNING_COLOR,
        _ => SUBTEXT_COLOR,
    }
}

fn size_display(item: &RepoItem, folder_sizes: &HashMap<String, u64>) -> String {
    if item.is_dir() {
        folder_sizes.get(&item.path).map_or_else(
            || format!("{:>10}", ""),
            |size| format!("{:>10}", format_size(*size)),
        )
    } else {
        item.actual_size().map_or_else(
            || format!("{:>10}", "-"),
            |size| format!("{:>10}", format_size(size)),
        )
    }
}

fn progress_percentage(completed: usize, total: usize) -> usize {
    if total == 0 {
        0
    } else {
        completed.saturating_mul(100) / total
    }
}

fn help_key(label: &'static str, color: ratatui::style::Color) -> Span<'static> {
    Span::styled(
        label,
        Style::default().fg(color).add_modifier(Modifier::BOLD),
    )
}

fn help_sep() -> Span<'static> {
    Span::styled("  |  ", Style::default().fg(BORDER_COLOR))
}

fn format_size(size: u64) -> String {
    if size < 1024 {
        format!("{size} B")
    } else if size < 1024 * 1024 {
        format!("{:.1} KB", size as f64 / 1024.0)
    } else if size < 1024 * 1024 * 1024 {
        format!("{:.1} MB", size as f64 / (1024.0 * 1024.0))
    } else {
        format!("{:.1} GB", size as f64 / (1024.0 * 1024.0 * 1024.0))
    }
}

fn get_file_type(name: &str, is_dir: bool) -> String {
    if is_dir {
        "DIR".to_string()
    } else {
        name.rsplit('.')
            .next()
            .filter(|ext| !ext.is_empty() && ext.len() <= 5)
            .map(|ext| ext.to_uppercase())
            .unwrap_or_else(|| "FILE".to_string())
    }
}

fn icon_for_item(item: &RepoItem) -> &'static str {
    if item.is_dir() {
        "▸ "
    } else if item.is_lfs() {
        "◉ "
    } else {
        "• "
    }
}

fn item_icon(item: &RepoItem, ascii_mode: bool) -> &'static str {
    if ascii_mode {
        if item.is_dir() {
            "[D] "
        } else if item.is_lfs() {
            "[L] "
        } else {
            "[F] "
        }
    } else {
        icon_for_item(item)
    }
}

fn truncate_name(name: &str, max_len: usize) -> String {
    if name.len() <= max_len {
        return name.to_string();
    }
    if let Some(dot_pos) = name.rfind('.') {
        let ext = &name[dot_pos..];
        let available = max_len.saturating_sub(ext.len()).saturating_sub(2);
        if available > 0 && available < dot_pos {
            format!("{}..{}", &name[..available], ext)
        } else {
            format!("{}..", &name[..max_len.saturating_sub(2)])
        }
    } else {
        format!("{}..", &name[..max_len.saturating_sub(2)])
    }
}
