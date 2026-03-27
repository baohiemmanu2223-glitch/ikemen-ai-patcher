# Ikemen AI Patcher ⚡

![Version](https://img.shields.io/badge/version-v0.5-blue.svg)
![Platform](https://img.shields.io/badge/platform-Windows-lightgrey.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)

**Ikemen AI Patcher** is a powerful, automated desktop application designed to inject boss-level, adaptive AI logic into characters designed for the MUGEN and Ikemen GO engines. By performing deep AST-level analysis on character `.cns`, `.st`, and `.cmd` code, the tool generates hyper-optimized and conflict-free logic specifically adhering to modern AI combat standards (Xiangfei-ROTD LLM Template).

## 🚀 Features

- **Character Code Analysis**: Deep parsing engine that auto-categorizes Normals, Specials, and Hypers using precise `HitDef` heuristics.
- **Deep Variable Sweeping**: Automatically maps the engine's entire memory allowance (`var(0-59)`, `fvar(0-39)`, `sysvar(0-4)`), and tracks **Variable Usage Profiles** (Read/Write States) to assign memory specifically without stepping on character-specific triggers.
- **Adaptive 3-Layer Combat AI**: Generates self-adjusting AI behavior featuring Reaction Spacing, Frame-Aware Punishes, and `Deadlock Breakers` to handle point-blank logic errors.
- **Tag-Team & Simul Optimization**: Completely rewrites static targeted checks with dynamic `enemynear(var(57))` pointers to ensure AI remains perfectly tracked against multiple partners.
- **ZSS Universal AI Styles Engine**: Allows users to import custom Universal `.zss` files utilizing abstract concepts (like `value = @projectile`) mapping directly to the underlying engine's evaluated states! Includes 7 pre-bundled styles (Aggressive, Zoning, Footsies, Defensive, Grappler, Mixup, Boss).
- **Syntax Auto-Sanitization**: Automatically scrubs any trigger-less logic dumps or legacy infinite loop syntax (e.g., `[State -1, Tick Fix]`) to avert total engine crashes!
- **Wails Desktop GUI**: Polished Desktop Frontend offering realtime insights, Combo Chain visuals, and Risk Safety validations in an interactive Dashboard.

---

## 🛠️ Installation & Build

This utility relies on [Wails v2](https://wails.io/) for bundling the Go Backend with the Frontend UI.

### Pre-requisites
- [Go](https://golang.org/dl/) (version 1.21+)
- [Wails](https://wails.io/docs/gettingstarted/installation) CLI

### Compilation
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/ikemen-ai-patcher.git
   cd ikemen-ai-patcher
   ```
2. Build the application for Windows:
   ```bash
   wails build -clean
   ```
3. The executable output will be located at `build/bin/ikemen-ai-patcher.exe`.

---

## 🖥️ Overview

### How to use
1. Launch `ikemen-ai-patcher.exe`.
2. Click **Select Character Folder** to load the MUGEN character repository. 
3. Click **Analyze Character** to parse the `cmd` and `cns` definitions and review the detailed variable tracking usage on the dashboard.
4. Head to **Styles** to select a Universal AI style, or remain on "Auto-Generated" for robust Boss heuristics.
5. Place the exe file in the same directory as the styles/ folder; the .zss files are located within this styles/ folder.
6. In **Patch Configuration**, adjust the Randomness and Anti-Spam variables, then hit **Patch AI**.
7. Your character files are safely backed up with `.bak`, and automatically updated with seamless integration logic wrapped between `IKEMEN-AI-PATCHER START` tags!

### Developer Contribution
Refer to our complete implementation [Architecture Guide (ai-patcher.md)](./ai-patcher.md) for technical specifications regarding to the AST Parser, ZSS expansion rules, and Variable standards.

---

## 🤝 Credits & Acknowledgements

- **Author**: cuocsong8x
- **Version**: v0.5
- **Contact**: Telegram huylong22
- **Donation**: If you find this software helpful, consider making a donation!
- **Core Technology**: Powered by Go, HTML/JS/CSS, Wails v2.
