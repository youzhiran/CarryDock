param(
    [string]$ChangelogPath = "CHANGELOG.md",
    [string]$Tag = "",
    [string]$Version = "",
    [string]$BuildNum = "",
    [string]$BuildTime = "",
    [string]$OutputPath = "dist/release-notes.md"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$unreleasedZh = -join [char[]]@(0x672A, 0x53D1, 0x5E03)

function Normalize-VersionKey {
    param([string]$Raw)
    if ([string]::IsNullOrWhiteSpace($Raw)) {
        return ""
    }
    $s = $Raw.Trim()
    if ($s -match '^\[([^\]]+)\]') {
        $s = $Matches[1].Trim()
    } else {
        $s = ($s -split '\s+-\s+|\s+', 2)[0].Trim()
    }
    if ($s -match '^[vV](?=\d)') {
        $s = $s.Substring(1)
    }
    return $s
}

function Strip-HtmlCommentsAndTrim {
    param([string[]]$Lines)
    if (-not $Lines -or $Lines.Count -eq 0) {
        return ""
    }
    $joined = [string]::Join("`n", $Lines)
    $withoutComments = [regex]::Replace($joined, '<!--[\s\S]*?-->', '')
    $cleanLines = $withoutComments -split "`r?`n"

    $start = 0
    while ($start -lt $cleanLines.Count -and [string]::IsNullOrWhiteSpace($cleanLines[$start])) {
        $start++
    }
    $end = $cleanLines.Count - 1
    while ($end -ge $start -and [string]::IsNullOrWhiteSpace($cleanLines[$end])) {
        $end--
    }
    if ($start -gt $end) {
        return ""
    }
    return ([string]::Join("`n", $cleanLines[$start..$end])).Trim()
}

$sections = @()
if (Test-Path -LiteralPath $ChangelogPath) {
    $allLines = [System.IO.File]::ReadAllLines((Resolve-Path -LiteralPath $ChangelogPath).Path, $utf8NoBom)
    $currentKey = $null
    $currentHeading = $null
    $currentLines = New-Object System.Collections.Generic.List[string]

    foreach ($line in $allLines) {
        if ($line -match '^##\s+(.+?)\s*$') {
            if ($null -ne $currentKey) {
                $sections += [PSCustomObject]@{
                    Key     = $currentKey
                    Heading = $currentHeading
                    Body    = Strip-HtmlCommentsAndTrim -Lines $currentLines.ToArray()
                }
            }
            $currentHeading = $Matches[1].Trim()
            $currentKey = Normalize-VersionKey -Raw $currentHeading
            $currentLines.Clear()
        } elseif ($null -ne $currentKey) {
            $currentLines.Add($line)
        }
    }
    if ($null -ne $currentKey) {
        $sections += [PSCustomObject]@{
            Key     = $currentKey
            Heading = $currentHeading
            Body    = Strip-HtmlCommentsAndTrim -Lines $currentLines.ToArray()
        }
    }
}

$cleanTag = Normalize-VersionKey -Raw $Tag
$cleanVer = Normalize-VersionKey -Raw $Version
$fullVer = if (-not [string]::IsNullOrWhiteSpace($cleanVer) -and -not [string]::IsNullOrWhiteSpace($BuildNum)) {
    "$cleanVer+$BuildNum"
} else {
    $cleanVer
}

$matchedSection = $null

# 1. Exact match against full tag or Version+BuildNum
$exactCandidates = @()
if (-not [string]::IsNullOrWhiteSpace($cleanTag)) { $exactCandidates += $cleanTag }
if (-not [string]::IsNullOrWhiteSpace($fullVer)) { $exactCandidates += $fullVer }
foreach ($cand in $exactCandidates) {
    $hit = $sections | Where-Object { $_.Key -ieq $cand -and -not [string]::IsNullOrWhiteSpace($_.Body) } | Select-Object -First 1
    if ($hit) {
        $matchedSection = $hit
        break
    }
}

# 2. Match short version (exact Version or unique Version+* entry)
if (-not $matchedSection -and -not [string]::IsNullOrWhiteSpace($cleanVer)) {
    $hit = $sections | Where-Object { $_.Key -ieq $cleanVer -and -not [string]::IsNullOrWhiteSpace($_.Body) } | Select-Object -First 1
    if (-not $hit) {
        $prefix = "$cleanVer+"
        $prefixHits = @($sections | Where-Object { $_.Key.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase) -and -not [string]::IsNullOrWhiteSpace($_.Body) })
        if ($prefixHits.Count -eq 1) {
            $hit = $prefixHits[0]
        }
    }
    if ($hit) {
        $matchedSection = $hit
    }
}

# 3. Fallback to non-empty Unreleased section
if (-not $matchedSection) {
    $matchedSection = $sections | Where-Object {
        ($_.Key -ieq "unreleased" -or $_.Key -eq $unreleasedZh) -and -not [string]::IsNullOrWhiteSpace($_.Body)
    } | Select-Object -First 1
}

$displayVersion = if (-not [string]::IsNullOrWhiteSpace($fullVer)) {
    $fullVer
} elseif (-not [string]::IsNullOrWhiteSpace($cleanTag)) {
    $cleanTag
} else {
    "dev"
}

$metaLines = New-Object System.Collections.Generic.List[string]
$metaLines.Add("- Version: $displayVersion")
if (-not [string]::IsNullOrWhiteSpace($BuildTime)) {
    $metaLines.Add("- Build Time: $BuildTime")
}

if ($matchedSection) {
    $finalContent = $matchedSection.Body + "`n`n---`n`n### Build Info`n" + ([string]::Join("`n", $metaLines)) + "`n"
} else {
    $finalContent = "CarryDock Windows Release (MyGo Native UI)`n" + ([string]::Join("`n", $metaLines)) + "`n"
}

$outDir = Split-Path -Parent $OutputPath
if (-not [string]::IsNullOrWhiteSpace($outDir) -and -not (Test-Path -LiteralPath $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}
[System.IO.File]::WriteAllText($OutputPath, $finalContent, $utf8NoBom)
$matchedLabel = if ($matchedSection) { $matchedSection.Key } else { "fallback" }
Write-Host ("Generated release notes at {0} (matched={1})" -f $OutputPath, $matchedLabel)
