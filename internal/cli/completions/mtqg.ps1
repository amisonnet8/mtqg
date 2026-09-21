# PowerShell completion for mtqg.
#
# It has no list of commands. At every TAB it asks `mtqg candidates` what can
# come next, so it is never out of date with the mtqg that is installed.
#
# Install it with:  mtqg completion powershell >> $PROFILE
# or try it with:   mtqg completion powershell | Out-String | Invoke-Expression
Register-ArgumentCompleter -Native -CommandName mtqg -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    # The words before the one being typed, without "mtqg": the elements of the
    # line that end before the cursor. The word being typed ends at the cursor
    # and is passed as --word: an empty word would be dropped, and "--word=" never is.
    $words = @($commandAst.CommandElements | Select-Object -Skip 1 |
        Where-Object { $_.Extent.EndOffset -lt $cursorPosition } |
        ForEach-Object {
            if ($_ -is [System.Management.Automation.Language.StringConstantExpressionAst]) { $_.Value } else { $_.Extent.Text }
        })

    $exitCode = $global:LASTEXITCODE
    $lines = mtqg candidates "--word=$wordToComplete" -- @words 2>$null
    $global:LASTEXITCODE = $exitCode

    foreach ($line in $lines) {
        if (-not $line) { continue }
        $value, $description = $line -split "`t", 2
        # A completion result must have a description to show.
        if (-not $description) { $description = $value }
        [System.Management.Automation.CompletionResult]::new($value, $value, 'ParameterValue', $description)
    }
}
