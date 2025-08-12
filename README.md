# Simple RISC-V Linker

This is a simple linker implementation that helps me learn how a linker works.

---

## ELF File Format Overview

<img src="images/ELF_overview.png.png" width="400">
*Image source: [ics.uci.edu](https://ics.uci.edu/~aburtsev/238P/hw/hw3-elf/hw3-elf.html)*

### 1. What is ELF?
- **ELF (Executable and Linkable Format)** is the file format used for `.o` object files in Linux.

### 2. Tools that Use ELF Files
- **Linker**: Combines multiple ELF files into an executable or a library.
- **Loader**: Loads the executable ELF file into the memory of a process.

### 3. Linker Requirements
- The linker needs to know the locations of sections like:
    - **DATA**
    - **TEXT**
    - **BSS**
    - Other relevant sections
- This information is necessary to merge them with sections from other libraries.

### 4. Loader Requirements
- The loader does not need section-level details.
- It only needs to know:
    - Which parts of the ELF file are **code** (executable).
    - Which parts are **data** and **read-only data**.
    - Where to place the **BSS** section in process memory.
- These permission restrictions are used to setup the page table entries while memory mapping in loader.

---

## ELF Header (**Ehdr**)

**Purpose:**  
The ELF header is the very first structure in an ELF file.  
It describes global properties of the file and tells both the loader and tools where to find other tables (program headers, section headers).

**Key fields:**
| Field       | Purpose |
|-------------|---------|
| `Ident`     | First 16 bytes, includes magic (`0x7F 'E' 'L' 'F'`) |
| `Machine`   | Target architecture (set this to `EM_RISCV`). |
| `Version`   | ELF version (usually `EV_CURRENT`). |
| `Entry`     | Entry point address |
| `Phoff`     | File offset of the program header table. |
| `Shoff`     | File offset of the section header table. |
| `Phentsize` / `Phnum` | Size and number of program header entries. |
| `Shentsize` / `Shnum` | Size and number of section header entries. |
| `Shstrndx`  | Index of `.shstrtab` in the section header table (section name string table). |

---

## Section Header (**Shdr**)

**Purpose:**  
Each section in the ELF file has a section header entry.  
These headers are **not used by the OS loader** at runtime but are essential for linkers and other tools to locate and interpret sections.

**Key fields:**
| Field       | Purpose |
|-------------|---------|
| `Name`      | Offset into `.shstrtab` (section name string table). |
| `Type`      | Section type |
| `Flags`     | Section attributes |
| `Addr`      |  |
| `Off`       | File offset of the section's data. |
| `Size`      | Size of the section in bytes. |
| `Link` / `Info` | |
| `Addralign` | Required memory/file alignment. |
| `Entsize`   | Size of each entry in the section (used for tables like `.symtab`). |

---

## ELF Section Header — `Type` vs `Flags`

### **Type**
- **Meaning:** Describes *what kind of data* the section contains and how tools should interpret it.
- **Common values:**
  - `SHT_PROGBITS` — Raw program data (e.g., `.text`, `.data`, `.rodata`).
  - `SHT_NOBITS` — No data in file, just memory space at runtime (e.g., `.bss`).
  - `SHT_SYMTAB` — Symbol table.
  - `SHT_STRTAB` — String table.
  - `SHT_RELA` / `SHT_REL` — Relocation entries.

### **Flags**
- **Meaning:** Describes *how the section behaves* and certain storage properties.
- **Common values:**
  - `SHF_ALLOC` — Section should be loaded into memory at runtime.
  - `SHF_EXECINSTR` — Contains executable code.
  - `SHF_WRITE` — Writable at runtime.
  - `SHF_MERGE` — Mergeable constants (identical fragments can be merged).
  - `SHF_STRINGS` — Contains NUL-terminated strings.

---

