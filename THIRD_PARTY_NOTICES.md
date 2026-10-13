# Attribution and licenses

MyCast is distributed under GNU Affero General Public License version 3 or, at your option, any later version (AGPL-3.0-or-later). Copyright (C) 2026 MyCast contributors. See LICENSE.

## Tinycast

- Source: https://github.com/abue-ammar/tinycast
- Copyright (C) 2026 Abue Ammar
- License: AGPL-3.0-or-later
- Used as a feature and architecture reference for a keyboard-first launcher, clipboard history, and separating pure logic from platform effects. No Tinycast Swift source has been copied or line-by-line translated in this first implementation. MyCast intentionally retains the same project license family.

## ZTools reference

- Sources: https://github.com/ZToolsCenter/ZTools/blob/main/scripts/compare-windows-scanner.cjs and https://github.com/ZToolsCenter/ZTools/blob/main/src/main/core/commandScanner/windowsScanner.ts
- License: MIT; Copyright (c) 2025 lzx8589561
- The documented desktop.ini/localized-name behavior was read as a reference.
  MyCast independently implements a bounded Go parser and resource-data reader;
  no ZTools native binary or dependency is bundled/executed, and its name/target
  deduplication is not adopted. Reference license text: [licenses/ZTools-MIT.txt](licenses/ZTools-MIT.txt).

## MyGo dependency

- Source: https://github.com/egoist/mygo
- Version: v0.4.6 (commit bc111b6d2cca221716ad99ddea1580449ad8c18e)
- License: MIT; Copyright (c) 2026 MyGo contributors
- Native UI and platform integration dependency. Lifecycle/tray usage follows its documented public APIs and examples. Original dependency license text is included below.

MIT License

Copyright (c) 2026 MyGo contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## go-pinyin and pinyin data

- Source: https://github.com/mozillazg/go-pinyin
- Version: v0.21.0 (commit db7796a07e7b9ac1284efe5dd657851d41ac2242)
- License: MIT; Copyright (c) 2016 mozillazg
- Bundled offline name transliteration only. Upstream uses pinyin-data
  (https://github.com/mozillazg/pinyin-data), also MIT, Copyright (c) 2016 mozillazg.
  Original license is retained in `licenses/go-pinyin-MIT.txt`.

## Transitive dependencies

Pinned versions and integrity hashes are recorded in go.mod and go.sum. They retain their original licenses; the MIT/BSD/Apache dependency licenses do not change MyCast's AGPL license.

Full dependency notices are preserved in `licenses/` and included in the Windows artifact.
