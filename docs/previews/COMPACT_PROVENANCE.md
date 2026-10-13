# Synthetic compact palette previews

All `compact-*-100/150/200.png` files are MyGo headless renders at 100%/150%/200%
scale from `panel_windows_test.go`, with dedicated synthetic application names,
paths and icon tiles. They are not desktop screenshots. Empty compact images
contain no inventory; expanded/control scenes use only the synthetic fixtures.

- `compact-empty`: opted-in empty call; no result widgets, search/footer retained.
- `compact-expanded`: Down explicitly expands and selects first visible result.
- `compact-settings`: full settings with the ordinary compact preference.
- `compact-constrained-menu`: 320-DIP expanded viewport, keyboard-selected final
  item visible after menu scroll.
- `compact-constrained-settings`: 320-DIP viewport with scrolled final controls
  and fixed accessible Save/Cancel actions.

Native Acrylic/corners, multi-monitor placement and actual keyboard/IME behavior
are not demonstrated by these images. An owned hidden native/MyGo window fixture
separately checks geometry/ownership/resource contracts without screen capture.
