# bash completion for mtqg.
#
# It has no list of commands. At every TAB it asks `mtqg candidates` what can
# come next, so it is never out of date with the mtqg that is installed.
#
# Install it with:  mtqg completion bash > ~/.local/share/bash-completion/completions/mtqg
# or try it with:   source <(mtqg completion bash)
_mtqg() {
    local value
    COMPREPLY=()
    # The words before the one being typed, and that one as --word: an empty
    # word would be dropped by some shells, and "--word=" never is.
    while IFS=$'\t' read -r value _; do
        COMPREPLY+=("$value")
    done < <(command mtqg candidates "--word=${COMP_WORDS[COMP_CWORD]}" -- "${COMP_WORDS[@]:1:COMP_CWORD-1}" 2>/dev/null)
}

# -o default: where nothing is offered (after -C, for example), files are.
complete -o default -F _mtqg mtqg
