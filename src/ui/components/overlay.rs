use ratatui::{
    Frame,
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Modifier, Style},
    text::{Line, Span, Text},
    widgets::{Block, Borders, Clear, Paragraph, Wrap},
};

use crate::ui::theme::*;
use crate::ui::{FilePreview, PreviewStatus, SavePrompt};

pub fn render_preview(f: &mut Frame, area: Rect, preview: &FilePreview) {
    let popup = centered_rect(88, 84, area);
    let inner = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(6), Constraint::Length(1)])
        .margin(1)
        .split(popup);

    let title = match preview.status {
        PreviewStatus::Loading => format!(" Previewing {} ", preview.path),
        PreviewStatus::Ready => format!(" Preview :: {} ", preview.path),
        PreviewStatus::Error => format!(" Preview Unavailable :: {} ", preview.path),
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(Span::styled(
            title,
            Style::default()
                .fg(ACCENT_COLOR)
                .add_modifier(Modifier::BOLD),
        ))
        .border_style(Style::default().fg(BORDER_COLOR))
        .style(Style::default().bg(OVERLAY_BG));

    let body = match &preview.status {
        PreviewStatus::Loading => Text::from(vec![
            Line::from(""),
            Line::from(Span::styled(
                "Loading file preview...",
                Style::default()
                    .fg(SUBTEXT_COLOR)
                    .add_modifier(Modifier::ITALIC),
            )),
        ]),
        PreviewStatus::Ready => {
            if preview.content.is_empty() {
                Text::from("(empty file)")
            } else {
                Text::from(preview.content.clone())
            }
        }
        PreviewStatus::Error => Text::from(vec![
            Line::from(""),
            Line::from(Span::styled(
                &preview.content,
                Style::default().fg(ERROR_COLOR),
            )),
        ]),
    };

    let content = Paragraph::new(body)
        .block(block)
        .wrap(Wrap { trim: false })
        .scroll((preview.scroll as u16, 0))
        .style(Style::default().fg(FG_COLOR).bg(OVERLAY_BG));

    let footer = Paragraph::new(Line::from(vec![
        Span::styled(
            "Esc",
            Style::default()
                .fg(ERROR_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(" close", Style::default().fg(SUBTEXT_COLOR)),
        Span::styled("  |  ", Style::default().fg(BORDER_COLOR)),
        Span::styled(
            "j/k",
            Style::default()
                .fg(ACCENT_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(" scroll", Style::default().fg(SUBTEXT_COLOR)),
        Span::styled("  |  ", Style::default().fg(BORDER_COLOR)),
        Span::styled(
            "d",
            Style::default()
                .fg(SUCCESS_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(" download from browser", Style::default().fg(SUBTEXT_COLOR)),
    ]))
    .alignment(Alignment::Center)
    .style(Style::default().bg(OVERLAY_BG));

    f.render_widget(Clear, popup);
    f.render_widget(content, popup);
    f.render_widget(footer, inner[1]);
}

pub fn render_save_prompt(f: &mut Frame, area: Rect, prompt: &SavePrompt) {
    let popup = centered_rect(72, 28, area);
    let inner = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(2),
            Constraint::Length(3),
            Constraint::Length(2),
            Constraint::Min(0),
        ])
        .margin(1)
        .split(popup);

    let display_input = if prompt.cursor >= prompt.input.len() {
        format!("{}_", prompt.input)
    } else {
        let mut s = prompt.input.clone();
        s.replace_range(prompt.cursor..prompt.cursor + 1, "_");
        s
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(Span::styled(
            " Save Download ",
            Style::default()
                .fg(ACCENT_COLOR)
                .add_modifier(Modifier::BOLD),
        ))
        .border_style(Style::default().fg(ACCENT_COLOR))
        .style(Style::default().bg(SURFACE_COLOR));

    let message = Paragraph::new(Line::from(vec![
        Span::styled(
            format!("{} item(s)", prompt.item_count),
            Style::default()
                .fg(SUCCESS_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            " will be saved into this directory. Default is the current working directory.",
            Style::default().fg(SUBTEXT_COLOR),
        ),
    ]))
    .style(Style::default().bg(SURFACE_COLOR));

    let input = Paragraph::new(display_input)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .title(Span::styled(
                    " Directory Path ",
                    Style::default()
                        .fg(FOLDER_COLOR)
                        .add_modifier(Modifier::BOLD),
                ))
                .border_style(Style::default().fg(BORDER_COLOR))
                .style(Style::default().bg(BG_COLOR)),
        )
        .style(Style::default().fg(FG_COLOR).bg(BG_COLOR));

    let footer = Paragraph::new(Line::from(vec![
        Span::styled(
            "Enter",
            Style::default()
                .fg(SUCCESS_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(" confirm", Style::default().fg(SUBTEXT_COLOR)),
        Span::styled("  |  ", Style::default().fg(BORDER_COLOR)),
        Span::styled(
            "Esc",
            Style::default()
                .fg(ERROR_COLOR)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(" cancel", Style::default().fg(SUBTEXT_COLOR)),
    ]))
    .alignment(Alignment::Center)
    .style(Style::default().bg(SURFACE_COLOR));

    f.render_widget(Clear, popup);
    f.render_widget(block, popup);
    f.render_widget(message, inner[0]);
    f.render_widget(input, inner[1]);
    f.render_widget(footer, inner[2]);
}

fn centered_rect(percent_x: u16, percent_y: u16, area: Rect) -> Rect {
    let vertical = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Percentage((100 - percent_y) / 2),
            Constraint::Percentage(percent_y),
            Constraint::Percentage((100 - percent_y) / 2),
        ])
        .split(area);

    Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage((100 - percent_x) / 2),
            Constraint::Percentage(percent_x),
            Constraint::Percentage((100 - percent_x) / 2),
        ])
        .split(vertical[1])[1]
}
