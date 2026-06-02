# Tool Usage Guide for LightBoot Project

## single_find_and_replace — The Correct Way

### CRITICAL: Arguments are literal, NOT escaped

The `old_string` and `new_string` arguments are taken **exactly as-is** — no escape processing happens.
Do NOT wrap in extra quotes. Do NOT use \n or \t escape sequences.

### Wrong (fails):

The tool wraps in outer quotes and uses \n \t escapes:
```
BEGIN_ARG: old_string
"func Foo() {\n\tbar()\n}"