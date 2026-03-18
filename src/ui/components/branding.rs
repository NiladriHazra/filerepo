use ratatui::{
    Frame,
    layout::Alignment,
    style::{Modifier, Style},
    text::{Line, Span},
    widgets::Paragraph,
};

use crate::ui::theme::{ACCENT_COLOR, BG_COLOR, BORDER_COLOR};

pub const APP_VERSION: &str = env!("CARGO_PKG_VERSION");

pub fn render_logo(f: &mut Frame, area: ratatui::layout::Rect) {
    let logo = [
        " ███████╗██╗██╗     ███████╗██████╗ ███████╗██████╗  ██████╗ ",
        " ██╔════╝██║██║     ██╔════╝██╔══██╗██╔════╝██╔══██╗██╔═══██╗",
        " █████╗  ██║██║     █████╗  ██████╔╝█████╗  ██████╔╝██║   ██║",
        " ██╔══╝  ██║██║     ██╔══╝  ██╔══██╗██╔══╝  ██╔═══╝ ██║   ██║",
        " ██║     ██║███████╗███████╗██║  ██║███████╗██║     ╚██████╔╝",
        " ╚═╝     ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚═╝      ╚═════╝ ",
    ];

    let lines = logo
        .into_iter()
        .map(|line| {
            Line::from(Span::styled(
                line,
                Style::default()
                    .fg(ACCENT_COLOR)
                    .add_modifier(Modifier::BOLD),
            ))
        })
        .collect::<Vec<_>>();

    let header = Paragraph::new(lines)
        .alignment(Alignment::Center)
        .style(Style::default().bg(BG_COLOR));
    f.render_widget(header, area);
}

pub fn version_line() -> Line<'static> {
    Line::from(Span::styled(
        format!("v{APP_VERSION}"),
        Style::default().fg(BORDER_COLOR),
    ))
}
