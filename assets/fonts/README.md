# Bundled fonts

PocketClaw bundles its typefaces rather than fetching them. `google_fonts`
downloaded Inter and Fira Code from Google's servers on first use, which made
ordinary UI rendering depend on the network and on a Google endpoint. That is
wrong for an app whose claim is that it runs on your phone, and it is
disqualifying for official F-Droid, which does not accept runtime downloads of
unpackaged assets.

Both families are licensed **SIL Open Font License 1.1**, which permits
redistribution as part of a bundled application. The licence texts are beside
the fonts and must stay there.

## Inter

- Upstream: <https://github.com/rsms/inter> release `v4.1`
- Archive: `Inter-4.1.zip`
  `9883fdd4a49d4fb66bd8177ba6625ef9a64aa45899767dde3d36aa425756b11e`
- Licence: [`Inter-OFL.txt`](Inter-OFL.txt) — OFL-1.1, © The Inter Project Authors
- Files taken from `extras/ttf/` in that archive

| File | Weight | SHA-256 |
| --- | --- | --- |
| `Inter-Regular.ttf` | 400 | `40d692fce188e4471e2b3cba937be967878f631ad3ebbbdcd587687c7ebe0c82` |
| `Inter-Medium.ttf` | 500 | `97ad806f526e41546d46365bb3a393145f75b7b1568913db74549ad8b8dba872` |
| `Inter-SemiBold.ttf` | 600 | `78a843fade9d4612a5567302fb595b56976eb5fcebf4fea5a5912d638bafcde3` |
| `Inter-Bold.ttf` | 700 | `288316099b1e0a47a4716d159098005eef7c0066921f34e3200393dbdb01947f` |
| `Inter-ExtraBold.ttf` | 800 | `e6756ad5690b77606aa62249a7b420d9902d45cae4b0048a24911fd4324b0a22` |
| `Inter-Black.ttf` | 900 | `6342d3ea6dc088b43867f615e807d898adf100c93edb978b8e52c5eb71a264da` |

## Fira Code

- Upstream: <https://github.com/tonsky/FiraCode> release `6.2`
- Archive: `Fira_Code_v6.2.zip`
  `0949915ba8eb24d89fd93d10a7ff623f42830d7c5ffc3ecbf960e4ecad3e3e79`
- Licence: [`FiraCode-OFL.txt`](FiraCode-OFL.txt) — OFL-1.1, © The Fira Code Project Authors
- Files taken from `ttf/` in that archive

| File | Weight | SHA-256 |
| --- | --- | --- |
| `FiraCode-Regular.ttf` | 400 | `5992ab9640e2df491b2f609467b1de60e8bc39b2c28db184342a0592d98f6117` |
| `FiraCode-SemiBold.ttf` | 600 | `500c74eec6249b06d49aef922dd3e8fc754c70c3b3f7791cd7b1a09ca9a26140` |

## Which weights, and why these

The bundled set is what the app actually asks for. `AppFonts.interTextTheme`
applies Inter across the Material text theme (400/500), and explicit call sites
use 600, 700, 800 and 900. Fira Code is used at 400 and 600 for logs and
column-aligned readouts.

Adding a new weight to the UI means adding the file here **and** declaring it in
`pubspec.yaml`. There is no fetch path to fall back on any more, which is the
point: a missing weight is a visible design decision rather than a silent
network call.
