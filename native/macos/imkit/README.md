# macOS IMKit Glue

This folder is reserved for the macOS InputMethodKit bundle.

The contract must mirror the Windows TSF edge:

- Load the same Go C ABI library from `core/abi`
- Forward key events to the engine
- Render native candidate UI
- Commit selected text through IMKit
- Keep all input logic out of Objective-C/Swift

The Windows implementation is the first local target for this workspace because the current machine is Windows.
