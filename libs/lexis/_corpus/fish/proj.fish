#!/usr/bin/env fish
# proj.fish - jump between projects and set up their environment.

set -g proj_root $HOME/projects
set -g proj_recent_file $__fish_config_dir/proj_recent
set -q proj_max_recent; or set -g proj_max_recent 10

function __proj_list --description 'Every project directory'
    for dir in $proj_root/*/
        if test -d $dir/.git; or test -f $dir/justfile
            basename $dir
        end
    end
end

function __proj_remember --argument-names name
    set -l recent
    if test -f $proj_recent_file
        set recent (string match -v -- $name < $proj_recent_file)
    end
    printf '%s\n' $name $recent[1..(math $proj_max_recent - 1)] > $proj_recent_file
end

function __proj_env --argument-names dir
    # Load KEY=value lines from .env, skipping comments and blanks.
    test -f $dir/.env; or return 0
    while read -l line
        string match -qr '^\s*(#|$)' -- $line; and continue
        set -l pair (string split -m 1 '=' -- $line)
        if test (count $pair) -eq 2
            set -gx $pair[1] (string trim -c '"\'' -- $pair[2])
        else
            echo "proj: bad line in .env: $line" >&2
        end
    end < $dir/.env
end

function proj --description 'Go to a project'
    argparse h/help l/list 'e/editor=' n/no-env -- $argv
    or return 2

    if set -q _flag_help
        echo "Usage: proj [-l] [-e EDITOR] [-n] [NAME]"
        echo "  -l  list projects, most recent first"
        return 0
    end

    if set -q _flag_list
        set -l seen
        if test -f $proj_recent_file
            while read -l name
                echo (set_color green)$name(set_color normal)
                set -a seen $name
            end < $proj_recent_file
        end
        for name in (__proj_list)
            contains -- $name $seen; or echo $name
        end
        return 0
    end

    set -l name $argv[1]
    if test -z "$name"
        set name (__proj_list | fzf --height 40% --prompt 'project> ')
        or return 1
    end

    set -l dir $proj_root/$name
    if not test -d $dir
        set -l close (__proj_list | string match -e -- $name)
        switch (count $close)
            case 0
                echo "proj: no project called '$name'" >&2
                return 1
            case 1
                set dir $proj_root/$close[1]
                set name $close[1]
            case '*'
                echo "proj: '$name' could be: $close" >&2
                return 1
        end
    end

    cd $dir; or return
    __proj_remember $name
    set -q _flag_no_env; or __proj_env $dir

    if set -q _flag_editor
        command $_flag_editor . &
        disown
    end

    set -l branch (command git branch --show-current 2>/dev/null)
    set -l dirty (command git status --porcelain 2>/dev/null | count)
    if test $dirty -gt 0
        echo "$name on $branch, $dirty changed"
    else if test -n "$branch"
        echo "$name on $branch"
    end
end

complete -c proj -f -a '(__proj_list)'
complete -c proj -s l -l list -d 'List projects'
complete -c proj -s e -l editor -r -d 'Open in an editor'

abbr -a p proj
bind \cp 'proj; commandline -f repaint'

for n in (seq 1 3)
    set -l total (math "$n * 2.5 + 0x10")
    not test $total -lt 0; and continue
    break
end
