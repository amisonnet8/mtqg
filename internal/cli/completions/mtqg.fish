# fish completion for mtqg.
#
# It has no list of commands. At every TAB it asks `mtqg candidates` what can
# come next, so it is never out of date with the mtqg that is installed.
#
# Install it with:  mtqg completion fish > ~/.config/fish/completions/mtqg.fish
# or try it with:   mtqg completion fish | source

function __mtqg_candidates
    # The words before the one being typed, without "mtqg".
    set -l words (commandline -opc)
    set -e words[1]
    set -l cur (commandline -ct)
    # --word is the word being typed: an empty word would be dropped, and
    # "--word=" never is. It is quoted so that the argument is there even where
    # $cur is a list with nothing in it, which unquoted would take the whole
    # argument away.
    mtqg candidates "--word=$cur" -- $words 2>/dev/null
end

# Where a file is wanted: after -C, and for the file of format.
function __mtqg_wants_file
    set -l words (commandline -opc)
    set -e words[1]
    test (count $words) -ge 1; and test "$words[-1]" = -C; and return 0
    test "$words[1]" = format
end

complete -c mtqg -f -a '(__mtqg_candidates)'
complete -c mtqg -F -n __mtqg_wants_file
