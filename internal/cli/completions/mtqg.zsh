#compdef mtqg
# zsh completion for mtqg.
#
# It has no list of commands. At every TAB it asks `mtqg candidates` what can
# come next, so it is never out of date with the mtqg that is installed.
#
# Install it with:  mtqg completion zsh > "${fpath[1]}/_mtqg"
# or try it with:   source <(mtqg completion zsh)
_mtqg() {
    local line value
    local -a items
    # The words before the one being typed, and that one as --word: an empty
    # word would be dropped by some shells, and "--word=" never is.
    for line in "${(@f)$(command mtqg candidates "--word=${words[CURRENT]}" -- "${(@)words[2,CURRENT-1]}" 2>/dev/null)}"; do
        [[ -n $line ]] || continue
        # "value<TAB>description" is "value:description" for _describe, which
        # wants a colon in the value written as \: (there is none in ours).
        if [[ $line == *$'\t'* ]]; then
            value=${line%%$'\t'*}
            items+=("${value//:/\\:}:${line#*$'\t'}")
        else
            items+=("${line//:/\\:}")
        fi
    done
    if (( ${#items} )); then
        _describe -t candidates 'mtqg' items
    else
        # Where nothing is offered (after -C, for example), files are.
        _files
    fi
}

if [ "$funcstack[1]" = "_mtqg" ]; then
    _mtqg "$@"
else
    compdef _mtqg mtqg
fi
