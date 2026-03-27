# Ikemen AI Patcher - Architecture & Features (v0.5)

This document is intended for LLMs or developers contributing to `ikemen-ai-patcher`. It provides an overview of the core components, design decisions, and AI code generation rules.

## 1. Project Structure (Wails + Go)

`ikemen-ai-patcher` is built as a hybrid desktop application using the **Wails v2** framework.
- **Backend (`main.go`, `app.go`, subpackages)**: Written in Go. Handles parsing MUGEN scripts, advanced frame logic analysis, and injecting AI text.
- **Frontend (`frontend/`)**: Written in pure HTML/CSS/JS. Connects to Go functions via `window.go.main.App`. Contains an `about.json` fetched dynamically for the About tab.

## 2. Core Modules

### A. Parser (`parser/`)
Parses `.cmd`, `.cns`, `.st` into an AST structure (`CharacterData`), extracting `[Statedef]` blocks, `[State]` controllers, and their triggers/values.

### B. Analyzer (`analyzer/`)
Processes the parsed data to find:
- **Move Categorization**: Normals (200-250, 400-450), Specials (1000-2999), Hypers (3000-4999).
- **Advanced Categorization**: Uses `HitDef` attributes to group states into semantic tags array: `projectile`, `antiair`, `reversal`, `overhead`, `low`, `grab`, `dash`, `launcher`.
- **Variable Allocation**: Scans the character for occupied `var()`, `fvar()`, and `sysvar()`.
  - Performs a **deep sweep** across the full MUGEN engine variable limits: `var(0-59)`, `fvar(0-39)`, and `sysvar(0-4)`. 
  - Generates a **VarUsage Profile** for each accessed variable, actively tracking precisely which State IDs Read (used as triggers) or Set (modified) the variable logic to aid developers in understanding its function.
  - Automatically reserves specific unused vars/fvars for AI implementation based on the **Xiangfei-ROTD LLM Standands**:
    - `var(59)`: AI Level | `var(57)`: Target Index (Simul/Tag support)
    - `fvar(15)`: Guard Check | `fvar(16)`: Grab Cooldown
    - `fvar(17)`: Enemy X Velocity Prediction
    - `fvar(31)`: Zoning Tracking | `fvar(33)`: Enemy Jump Count

### C. AI Generator (`ai/`)
Generates the actual Ikemen GO logic (`[State -1]`) injected into `Cmd.cmd` based on the variables above.
- **Strict Syntax Rules**: All conditions output `triggerall = foo = 1` not `foo == 1`. 
- **Template Standard**: Uses the *Xiangfei-ROTD LLM template*, which ensures intelligent distance checks (Velocity Prediction), Deadlock breaking, Roll Evasion, and proper `enemynear(var(57))` targeting for robust Tag-team logic.
- **CNS Background Helpers**: Generates Dummy Statedefs (e.g., `Statedef 9740`, `Statedef 33333333`) into the `.cns` file so background helper AI tracking modules don't trigger Double/Clone glitches.

### D. Universal AI Styles (`style/`)
A custom configuration format (`.zss` files) residing in the `styles/` root directory.
- Users can define external logic decoupled from hardcoded `StateNo`.
- **Abstract Categories**: Instead of `value = 210`, a ZSS file can specify `value = @projectile`. 
- **Expansion Engine**: During patching, `ExpandAbstractStates` translates `@projectile` into multiple `AIDecision` nodes populated with exactly the State IDs discovered by the Analyzer.

### E. Patcher (`patcher/`)
Renders the `.bak` backups before modifying character source code.
- **Sanitization Layer**: Automatically scans and strips out `[State -1, Tick Fix]` loops from the legacy `.cmd` that cause logical crashes.
- **Triggerless Block GC**: Automatically scrubs the generated AI text output to remove blocks missing `trigger1` or `triggerall` conditions before writing, averting Ikemen engine fatal crashes.

## 3. Dealing With Conflicts (For Future LLMs)

If asked to fix a bug or add a feature, adhere to the following:
1. **IKEMEN EQUALITY**: Never use `==` in any generated MUGEN code string. Use `=`, `!=`, `>=`, `<=`.
2. **NO HARDCODING TARGETS**: When writing `EnemyNear`, always use `enemynear(var(57))` (the target index var pointer) when it exists in memory allocation to correctly support Simul/Tag mode targets.
3. **AVOID `State -1` TRIGGERLESS BLOCKS**: If a state block does not contain `trigger...`, it will crash Ikemen. The `Patcher` layer filters them out, but generation should ideally not produce them.
4. **STYLE PARSER MAPS**: Do not assume `currentDecision.State > 0` directly in `style/parser.go`, because abstract decisions (e.g. `@antiair`) will have `State == 0` until unrolled downstream.
