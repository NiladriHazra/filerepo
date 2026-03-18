use ratatui::{
    Frame,
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Modifier, Style},
    text::Span,
    widgets::{Block, Borders, Paragraph},
};

use crate::ui::components::branding;
use crate::ui::theme::*;

pub fn render(f: &mut Frame, area: Rect, frame_count: u64, status_msg: &str) {
    let vertical_layout = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Percentage(15),
            Constraint::Length(8),
            Constraint::Length(2),
            Constraint::Length(3),
            Constraint::Length(2),
            Constraint::Min(0),
        ])
        .split(area);

    let header_area = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(10),
            Constraint::Percentage(80),
            Constraint::Percentage(10),
        ])
        .split(vertical_layout[1]);

    branding::render_logo(f, header_area[1]);

    let msg = if status_msg.is_empty() {
        "Loading repository..."
    } else {
        status_msg
    };
    let status = Paragraph::new(Span::styled(
        msg,
        Style::default()
            .fg(SUBTEXT_COLOR)
            .add_modifier(Modifier::ITALIC),
    ))
    .alignment(Alignment::Center)
    .style(Style::default().bg(BG_COLOR));
    f.render_widget(status, vertical_layout[2]);

    let spinner_frames = [
        "   .      ",
        "   ..     ",
        "   ...    ",
        "   ....   ",
        "    ...   ",
        "     ..   ",
        "      .   ",
        "          ",
    ];
    let frame_idx = (frame_count / 3) as usize % spinner_frames.len();

    let spinner_area = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Min(0),
            Constraint::Length(20),
            Constraint::Min(0),
        ])
        .split(vertical_layout[3]);

    let spinner = Paragraph::new(Span::styled(
        spinner_frames[frame_idx],
        Style::default()
            .fg(MAUVE_COLOR)
            .add_modifier(Modifier::BOLD),
    ))
    .alignment(Alignment::Center)
    .block(
        Block::default()
            .borders(Borders::NONE)
            .style(Style::default().bg(BG_COLOR)),
    );
    f.render_widget(spinner, spinner_area[1]);

    let hint = Paragraph::new(Span::styled(
        "Press ESC to cancel",
        Style::default().fg(BORDER_COLOR),
    ))
    .alignment(Alignment::Center)
    .style(Style::default().bg(BG_COLOR));
    f.render_widget(hint, vertical_layout[4]);
}
