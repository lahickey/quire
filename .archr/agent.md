Annotations say how the project is meant to be, and the code should follow them.
Comments are the human's questions and requests about the code.

When the human talks about "the review" without saying which, take WORKTREE.

The conversation with the human is where you talk a comment through.
A reply is the record of where it landed, read beside the code: what you answered, what was agreed, what you did.

- Work on the open ones. An answered one is waiting for the human; leave it until they reply, which makes it open again.
- Reply to every comment you take on. Then, in the conversation, give one line per comment: its path and what happened (answered, done, or what you need from the human). Never restate what the reply says; the human reads it in archr. For example:

      diagram/          answered: needs your pick between two shapes
      examples/         done
      cmd/quire/font.go raised in the conversation

- Keep a reply to a few sentences: the answer, or what you did. A review is for small, settled things.
- When a comment opens a design question (several options, trade-offs, research needed, or it touches several comments), do not argue it in the reply. Reply with one line, "Raised in the conversation", and take it there. Once it is settled, reply again with the outcome in a sentence or two. When the human says to take a comment to the conversation, do that.
- A question gets an answer as its reply, not a change.
- A request is confirmed with the human before you act on it, unless they have told you to go ahead. Agreement in the conversation counts: say so in the reply, as in "Done, as agreed in the conversation: moved the fonts to assets/fonts."
- A comment that is both, such as "surely this belongs in bake/?", gets an answer first, with the change offered.
- When one answer covers several comments, give it in full on one and point the others at it: "See k7mq2xfa."
- When an answer of yours turns out wrong, or a later change makes it stale, reply again to the same comment, saying what changed.
- When a comment asks for something an annotation rules out, the annotation wins: stop and ask the human. Never decide between them yourself.
- When it is unclear what a comment is about, say in the reply what you took it to mean.
- Close a review only when the human tells you to, then say in the conversation which you closed and what was kept.
- When a review you were working on has been closed, report what you did in the conversation instead.
- Reply as yourself, `--as claude`, adding what you are working on, as `claude-fonts`, only when the human runs more than one of you.

To show the human some code, such as a flow through several files, write a tour. Keep each stop's text short: it is drawn on the code.
When you run inside herdr (HERDR_ENV is 1), open it for them beside you:

    herdr pane split --current --direction right --cwd "$PWD" --no-focus
    herdr pane run PANE "archr tour ID"

with PANE the pane_id the split prints. Otherwise tell them the command.
