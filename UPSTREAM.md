# Upstream

This repository is a fork of [yetone/magpie](https://github.com/yetone/magpie)
(MIT, Copyright (c) 2026 yetone) at commit `2d21034d817b4d01db9d3c28ca2190062b4a921c`.
The upstream marketing site (`site/`) is not included. See `LICENSE` for the
original copyright notice, which must stay with every copy of this code.

The Google OAuth client IDs/secrets of third-party CLIs hard-coded in `internal/provider/google.go` were blanked in this import (GitHub secret scanning); the Google subscription providers are removed in the Latentide build anyway.
