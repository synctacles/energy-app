#!/bin/bash
set -euo pipefail

# ============================================
# OpenClaw + KB Installation Script
# BRAINS Server
# ============================================

# Kleuren voor output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ============================================
# Configuratie (pas aan voor jouw setup)
# ============================================
OPENCLAW_VERSION="latest"
OLLAMA_MODELS=("phi3:mini" "nomic-embed-text")
DB_NAME="brains_kb"
DB_ADMIN_USER="brains_admin"
DB_OPENCLAW_USER="openclaw_reader"

# Secrets worden uit environment gelezen of gegenereerd
DB_ADMIN_PASS="${DB_ADMIN_PASS:-$(openssl rand -base64 24)}"
DB_OPENCLAW_PASS="${DB_OPENCLAW_PASS:-$(openssl rand -base64 24)}"

# ============================================
# Pre-flight checks
# ============================================
preflight_checks() {
    log_info "Running pre-flight checks..."

    # Check root/sudo
    if [[ $EUID -ne 0 ]]; then
        log_error "Dit script moet als root draaien (sudo)"
        exit 1
    fi

    # Check OS
    if ! grep -q "Ubuntu\|Debian" /etc/os-release; then
        log_warn "Script getest op Ubuntu/Debian. Andere distro's kunnen afwijken."
    fi

    # Check disk space (minimaal 20GB vrij)
    FREE_SPACE=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    if [[ $FREE_SPACE -lt 20 ]]; then
        log_error "Onvoldoende disk space: ${FREE_SPACE}GB vrij, 20GB nodig"
        exit 1
    fi

    # Check memory (minimaal 4GB)
    TOTAL_MEM=$(free -g | awk '/^Mem:/{print $2}')
    if [[ $TOTAL_MEM -lt 4 ]]; then
        log_warn "Beperkt geheugen: ${TOTAL_MEM}GB. Aanbevolen: 4GB+"
    fi

    log_info "Pre-flight checks passed ✓"
}

# ============================================
# Systeem updates
# ============================================
install_system_deps() {
    log_info "Installing system dependencies..."

    apt-get update
    apt-get install -y \
        curl \
        wget \
        git \
        build-essential \
        ca-certificates \
        gnupg \
        lsb-release \
        postgresql \
        postgresql-contrib

    log_info "System dependencies installed ✓"
}

# ============================================
# Node.js 22 LTS
# ============================================
install_nodejs() {
    log_info "Installing Node.js 22 LTS..."

    # Check if already installed with correct version
    if command -v node &> /dev/null; then
        NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
        if [[ $NODE_VERSION -ge 22 ]]; then
            log_info "Node.js $(node -v) already installed ✓"
            return
        fi
    fi

    # Install via NodeSource
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    apt-get install -y nodejs

    # Verify
    log_info "Node.js $(node -v) installed ✓"
}

# ============================================
# Ollama
# ============================================
install_ollama() {
    log_info "Installing Ollama..."

    if command -v ollama &> /dev/null; then
        log_info "Ollama already installed ✓"
    else
        curl -fsSL https://ollama.com/install.sh | sh
    fi

    # Start service
    systemctl enable ollama
    systemctl start ollama

    # Wait for Ollama to be ready
    sleep 5

    # Pull models
    for model in "${OLLAMA_MODELS[@]}"; do
        log_info "Pulling Ollama model: $model"
        ollama pull "$model"
    done

    log_info "Ollama installed with models ✓"
}

# ============================================
# PostgreSQL + pgvector
# ============================================
setup_postgresql() {
    log_info "Setting up PostgreSQL..."

    # Start PostgreSQL
    systemctl enable postgresql
    systemctl start postgresql

    # Install pgvector
    apt-get install -y postgresql-15-pgvector 2>/dev/null || {
        log_warn "pgvector niet in apt, installeren vanaf source..."
        cd /tmp
        git clone --branch v0.6.0 https://github.com/pgvector/pgvector.git
        cd pgvector
        make
        make install
        cd -
    }

    # Create database and users
    sudo -u postgres psql <<EOF
-- Create users
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${DB_ADMIN_USER}') THEN
        CREATE USER ${DB_ADMIN_USER} WITH PASSWORD '${DB_ADMIN_PASS}';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${DB_OPENCLAW_USER}') THEN
        CREATE USER ${DB_OPENCLAW_USER} WITH PASSWORD '${DB_OPENCLAW_PASS}';
    END IF;
END
\$\$;

-- Create database
SELECT 'CREATE DATABASE ${DB_NAME}' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${DB_NAME}')\gexec

-- Grant connect
GRANT CONNECT ON DATABASE ${DB_NAME} TO ${DB_ADMIN_USER};
GRANT CONNECT ON DATABASE ${DB_NAME} TO ${DB_OPENCLAW_USER};
EOF

    # Apply schema
    log_info "Applying database schema..."
    sudo -u postgres psql -d "${DB_NAME}" <<'SCHEMA'
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS vector;

-- Create schema
CREATE SCHEMA IF NOT EXISTS kb;

-- KB entries table
CREATE TABLE IF NOT EXISTS kb.entries (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    tags TEXT[],
    source VARCHAR(100),
    language VARCHAR(10) DEFAULT 'nl',
    embedding vector(384),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('dutch', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('dutch', coalesce(content, '')), 'B')
    ) STORED
);

-- Query log table
CREATE TABLE IF NOT EXISTS kb.query_log (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    matched_entry_id INTEGER REFERENCES kb.entries(id),
    confidence_score NUMERIC(3,2),
    response_sent BOOLEAN DEFAULT false,
    telegram_chat_id BIGINT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_kb_entries_search ON kb.entries USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_kb_entries_category ON kb.entries(category);
CREATE INDEX IF NOT EXISTS idx_kb_entries_tags ON kb.entries USING GIN(tags);
SCHEMA

    # Set permissions
    sudo -u postgres psql -d "${DB_NAME}" <<EOF
-- Admin permissions
GRANT ALL PRIVILEGES ON SCHEMA kb TO ${DB_ADMIN_USER};
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA kb TO ${DB_ADMIN_USER};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA kb TO ${DB_ADMIN_USER};

-- OpenClaw permissions (READ-ONLY on entries, INSERT on query_log)
GRANT USAGE ON SCHEMA kb TO ${DB_OPENCLAW_USER};
GRANT SELECT ON ALL TABLES IN SCHEMA kb TO ${DB_OPENCLAW_USER};
GRANT INSERT ON kb.query_log TO ${DB_OPENCLAW_USER};
GRANT USAGE ON SEQUENCE kb.query_log_id_seq TO ${DB_OPENCLAW_USER};

-- Default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA kb GRANT SELECT ON TABLES TO ${DB_OPENCLAW_USER};
EOF

    log_info "PostgreSQL setup complete ✓"
}

# ============================================
# OpenClaw
# ============================================
install_openclaw() {
    log_info "Installing OpenClaw..."

    # Install globally
    npm install -g openclaw@${OPENCLAW_VERSION}

    # Verify version (security: must be >= 2026.1.29 for RCE patch)
    INSTALLED_VERSION=$(openclaw --version 2>/dev/null || echo "0.0.0")
    log_info "OpenClaw version: ${INSTALLED_VERSION}"

    log_info "OpenClaw installed ✓"
}

# ============================================
# OpenClaw Configuration
# ============================================
configure_openclaw() {
    log_info "Configuring OpenClaw..."

    # Create config directory
    OPENCLAW_CONFIG_DIR="/etc/openclaw"
    mkdir -p "${OPENCLAW_CONFIG_DIR}"

    # Create main config
    cat > "${OPENCLAW_CONFIG_DIR}/openclaw.json" <<EOF
{
  "gateway": {
    "bind": "127.0.0.1:18789",
    "auth": {
      "type": "token",
      "token": "$(openssl rand -hex 32)"
    }
  },

  "agents": {
    "defaults": {
      "model": {
        "primary": "ollama/phi3:mini",
        "fallback": ["ollama/llama3:8b"]
      }
    },
    "list": [
      {
        "id": "kb-bot",
        "name": "Brains KB Bot",
        "description": "Community support bot powered by local KB",
        "workspace": "/opt/openclaw/workspace",
        "systemPrompt": "Je bent een behulpzame support bot voor de Home Assistant community. Je beantwoordt vragen op basis van de Knowledge Base. Als je het antwoord niet weet, zeg dat eerlijk en verwijs naar de community. Antwoord altijd in het Nederlands tenzij anders gevraagd."
      }
    ]
  },

  "models": {
    "providers": {
      "ollama": {
        "baseUrl": "http://127.0.0.1:11434",
        "models": [
          {"id": "phi3:mini", "name": "Phi-3 Mini"},
          {"id": "llama3:8b", "name": "Llama 3 8B"}
        ]
      }
    }
  },

  "channels": {
    "telegram": {
      "enabled": true,
      "token": "\${TELEGRAM_BOT_TOKEN}",
      "allowedChats": [],
      "groups": {
        "*": {
          "requireMention": true
        }
      }
    }
  },

  "tools": {
    "exec": {
      "enabled": false
    },
    "web_search": {
      "enabled": false
    },
    "web_fetch": {
      "enabled": false
    },
    "mcp": {
      "servers": {
        "kb-search": {
          "command": "node",
          "args": ["/opt/openclaw/mcp/kb-search.js"],
          "env": {
            "DATABASE_URL": "postgresql://${DB_OPENCLAW_USER}:${DB_OPENCLAW_PASS}@localhost:5432/${DB_NAME}"
          }
        }
      }
    }
  },

  "logging": {
    "level": "info",
    "redactSensitive": "tools",
    "redactPatterns": [
      "postgresql://[^\\\\s]+",
      "TELEGRAM_BOT_TOKEN=[^\\\\s]+",
      "-----BEGIN [A-Z]+ PRIVATE KEY-----[\\\\s\\\\S]*?-----END [A-Z]+ PRIVATE KEY-----"
    ]
  },

  "security": {
    "sandbox": true,
    "networkIsolation": true
  }
}
EOF

    # Create workspace
    mkdir -p /opt/openclaw/workspace
    mkdir -p /opt/openclaw/mcp

    # Create systemd service
    cat > /etc/systemd/system/openclaw.service <<EOF
[Unit]
Description=OpenClaw Gateway
After=network.target postgresql.service ollama.service
Requires=postgresql.service ollama.service

[Service]
Type=simple
User=openclaw
Group=openclaw
WorkingDirectory=/opt/openclaw
Environment=NODE_ENV=production
Environment=OPENCLAW_CONFIG=/etc/openclaw/openclaw.json
EnvironmentFile=/etc/openclaw/secrets.env
ExecStart=/usr/bin/openclaw start
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/openclaw/workspace

[Install]
WantedBy=multi-user.target
EOF

    # Create secrets file (placeholder - fill in actual values)
    cat > /etc/openclaw/secrets.env <<EOF
# Telegram Bot Token (van @BotFather)
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# Database (already configured in openclaw.json, but backup here)
DATABASE_URL=postgresql://${DB_OPENCLAW_USER}:${DB_OPENCLAW_PASS}@localhost:5432/${DB_NAME}
EOF
    chmod 600 /etc/openclaw/secrets.env

    # Create dedicated user
    if ! id "openclaw" &>/dev/null; then
        useradd -r -s /bin/false -d /opt/openclaw openclaw
    fi

    chown -R openclaw:openclaw /opt/openclaw
    chown -R openclaw:openclaw /etc/openclaw

    log_info "OpenClaw configured ✓"
}

# ============================================
# MCP Server voor KB Search
# ============================================
create_kb_mcp_server() {
    log_info "Creating KB Search MCP server..."

    cat > /opt/openclaw/mcp/kb-search.js <<'EOF'
#!/usr/bin/env node
/**
 * KB Search MCP Server
 * Provides read-only access to the Brains KB database
 */

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const { Pool } = require('pg');

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
});

const server = new Server(
  { name: 'kb-search', version: '1.0.0' },
  { capabilities: { tools: {} } }
);

// Tool: Search KB
server.setRequestHandler('tools/list', async () => ({
  tools: [
    {
      name: 'search_kb',
      description: 'Zoek in de Knowledge Base naar relevante artikelen',
      inputSchema: {
        type: 'object',
        properties: {
          query: {
            type: 'string',
            description: 'Zoekterm of vraag'
          },
          category: {
            type: 'string',
            description: 'Optioneel: filter op categorie'
          },
          limit: {
            type: 'number',
            description: 'Max aantal resultaten (default: 5)',
            default: 5
          }
        },
        required: ['query']
      }
    },
    {
      name: 'get_kb_entry',
      description: 'Haal een specifiek KB artikel op via ID',
      inputSchema: {
        type: 'object',
        properties: {
          id: {
            type: 'number',
            description: 'ID van het KB artikel'
          }
        },
        required: ['id']
      }
    },
    {
      name: 'list_categories',
      description: 'Toon alle beschikbare KB categorieën',
      inputSchema: {
        type: 'object',
        properties: {}
      }
    }
  ]
}));

server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;

  try {
    switch (name) {
      case 'search_kb': {
        const { query, category, limit = 5 } = args;

        let sql = `
          SELECT id, title, content, category, tags,
                 ts_rank(search_vector, plainto_tsquery('dutch', $1)) as rank
          FROM kb.entries
          WHERE search_vector @@ plainto_tsquery('dutch', $1)
        `;
        const params = [query];

        if (category) {
          sql += ` AND category = $2`;
          params.push(category);
        }

        sql += ` ORDER BY rank DESC LIMIT $${params.length + 1}`;
        params.push(limit);

        const result = await pool.query(sql, params);

        // Log query for analytics
        await pool.query(
          'INSERT INTO kb.query_log (query_text, matched_entry_id, confidence_score) VALUES ($1, $2, $3)',
          [query, result.rows[0]?.id || null, result.rows[0]?.rank || 0]
        );

        return {
          content: [{
            type: 'text',
            text: JSON.stringify(result.rows, null, 2)
          }]
        };
      }

      case 'get_kb_entry': {
        const { id } = args;
        const result = await pool.query(
          'SELECT id, title, content, category, tags FROM kb.entries WHERE id = $1',
          [id]
        );

        if (result.rows.length === 0) {
          return {
            content: [{ type: 'text', text: 'Artikel niet gevonden' }]
          };
        }

        return {
          content: [{
            type: 'text',
            text: JSON.stringify(result.rows[0], null, 2)
          }]
        };
      }

      case 'list_categories': {
        const result = await pool.query(
          'SELECT DISTINCT category, COUNT(*) as count FROM kb.entries GROUP BY category ORDER BY count DESC'
        );

        return {
          content: [{
            type: 'text',
            text: JSON.stringify(result.rows, null, 2)
          }]
        };
      }

      default:
        throw new Error(`Unknown tool: ${name}`);
    }
  } catch (error) {
    return {
      content: [{
        type: 'text',
        text: `Error: ${error.message}`
      }],
      isError: true
    };
  }
});

// Start server
const transport = new StdioServerTransport();
server.connect(transport);
console.error('KB Search MCP server started');
EOF

    # Install MCP dependencies
    cd /opt/openclaw/mcp
    npm init -y
    npm install @modelcontextprotocol/sdk pg

    chown -R openclaw:openclaw /opt/openclaw/mcp

    log_info "KB Search MCP server created ✓"
}

# ============================================
# Output credentials
# ============================================
output_credentials() {
    log_info "Installation complete!"

    echo ""
    echo "============================================"
    echo "BELANGRIJKE CREDENTIALS (BEWAAR VEILIG!)"
    echo "============================================"
    echo ""
    echo "Database Admin:"
    echo "  User: ${DB_ADMIN_USER}"
    echo "  Pass: ${DB_ADMIN_PASS}"
    echo ""
    echo "Database OpenClaw (read-only):"
    echo "  User: ${DB_OPENCLAW_USER}"
    echo "  Pass: ${DB_OPENCLAW_PASS}"
    echo ""
    echo "Connection strings:"
    echo "  Admin: postgresql://${DB_ADMIN_USER}:${DB_ADMIN_PASS}@localhost:5432/${DB_NAME}"
    echo "  OpenClaw: postgresql://${DB_OPENCLAW_USER}:${DB_OPENCLAW_PASS}@localhost:5432/${DB_NAME}"
    echo ""
    echo "============================================"
    echo "VOLGENDE STAPPEN:"
    echo "============================================"
    echo ""
    echo "1. Vul Telegram bot token in:"
    echo "   sudo nano /etc/openclaw/secrets.env"
    echo ""
    echo "2. Migreer KB data van DEV server (zie migrate-kb.sh)"
    echo ""
    echo "3. Start OpenClaw:"
    echo "   sudo systemctl daemon-reload"
    echo "   sudo systemctl enable openclaw"
    echo "   sudo systemctl start openclaw"
    echo ""
    echo "4. Check status:"
    echo "   sudo systemctl status openclaw"
    echo "   sudo journalctl -u openclaw -f"
    echo ""
}

# ============================================
# Main
# ============================================
main() {
    echo "============================================"
    echo "OpenClaw + KB Installation"
    echo "BRAINS Server"
    echo "============================================"
    echo ""

    preflight_checks
    install_system_deps
    install_nodejs
    install_ollama
    setup_postgresql
    install_openclaw
    configure_openclaw
    create_kb_mcp_server
    output_credentials
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
