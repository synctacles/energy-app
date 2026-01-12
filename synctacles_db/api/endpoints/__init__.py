# Active endpoints
from . import balance
from . import prices
from . import energy_action

# Deprecated endpoints (410 Gone) - Phase 2/3: Soft Delete (2026-01-11)
# Modules moved to archive/ directory:
# - now.py (Phase 3: replaced by energy_action)
# - generation_mix.py (Phase 2: grid data discontinued)
# - load.py (Phase 2: grid data discontinued)
# - signals.py (Phase 2: grid data discontinued)
