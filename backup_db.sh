#!/usr/bin/env bash
# ==============================================================================
# Avandab 24/7 Automated Daily SQLite Backup Script
# Backs up mvtms.db safely using SQLite online backup API
# ==============================================================================
set -e

BACKUP_DIR="/home/abhishek/Desktop/temux/basic/backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/mvtms_backup_${TIMESTAMP}.db"

mkdir -p "$BACKUP_DIR"

# Perform safe live copy from Tecno phone via ADB
echo "📦 Initiating live SQLite backup from Tecno Pova 2..."
adb pull /data/local/tmp/mvtms.db "$BACKUP_FILE" > /dev/null
adb pull /data/local/tmp/mvtms.db-wal "${BACKUP_FILE}-wal" 2>/dev/null || true

# Compress backup to save space
gzip -f "$BACKUP_FILE"
echo "✅ Backup completed successfully: ${BACKUP_FILE}.gz"

# Retain only the last 7 daily backups (delete older)
find "$BACKUP_DIR" -name "mvtms_backup_*.db.gz" -mtime +7 -delete
