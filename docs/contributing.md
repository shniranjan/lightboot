site_name: LightBoot Documentation
site_description: PXE Network Boot Manager — Documentation
site_author: LightBoot Contributors
site_dir: site
repo_url: https://github.com/shniranjan/lightboot
edit_uri: edit/main/docs/

theme:
  name: material
  palette:
    scheme: slate
    primary: indigo
  features:
    - navigation.instant
    - navigation.tracking
    - navigation.tabs
    - navigation.sections
    - navigation.indexes
    - search.suggest
    - search.highlight
    - content.code.copy
    - toc.follow
  icon:

    logo: material/lightbulb
    repo: fontawesome/brands/github

markdown_extensions:
  - pymdownx.highlight:
      anchor_linenums: true
  - pymdownx.inlinehilite
  - pymdownx.superfences
  - pymdownx.tabbed:
      alternate_style: true
  - pymdownx.tasklist:
      custom_checkbox: true
  - admonition
  - footnotes
  - toc:
      permalink: true

nav:
  - Home: index.md
  - Getting Started:
      - Installation: installation.md
      - Configuration: configuration.md
  - Usage:
      - ISO Management: iso-management.md
      - Profiles: profiles.md
      - Boot Modes: boot-modes.md
  - Networking: networking.md
  - API Reference: api-reference.md
  - Advanced:
      - Secure Boot: secure-boot.md
      - Troubleshooting: troubleshooting.md
  - Contributing: contributing.md

plugins:
  - search
