#!/bin/sh

DESC=$1
DESC=${DESC// /_}
MIGRATION_PATH=$2

if [ -z "$DESC" ]; then
  echo "Usage: add-revision.sh <description> <migration_path>"
  echo "Arguments:"
  echo "    description       Migration description"
  echo "    migration_path    Migration directory path, default is v1 skygear-sever"
  echo ""
  echo "Example:"
  echo "    # Add migration revision to v1 skygear-server"
  echo "    ./add-revision.sh add_username"
  echo ""
  echo "    # Add migration revision to next gateway"
  echo "    ./add-revision.sh add_tenant_config ../pkg/gateway/db/migration"
  exit;
fi

BASEDIR=$(dirname "$0")
SKYDB_UUID=$(uuidgen | tr -d - | tr -d '\n' | tr '[:upper:]' '[:lower:]')
SKYDB_REV=${SKYDB_UUID: -12}
SKYDB_REV_FILENAME=${SKYDB_REV}_${DESC}.go
SKYDB_MIGRATION_PATH=${MIGRATION_PATH:=${BASEDIR}/../pkg/server/skydb/pq/migration}
SKYDB_REV_TEMPLATE=${SKYDB_MIGRATION_PATH}/revision.go.template
SKYDB_REV_FILE_PATH=${SKYDB_MIGRATION_PATH}/${SKYDB_REV_FILENAME}

cp ${SKYDB_REV_TEMPLATE} ${SKYDB_REV_FILE_PATH}
sed -i '.bak' -e "s/__VERSION__/${SKYDB_REV}/g" ${SKYDB_REV_FILE_PATH}
rm ${SKYDB_REV_FILE_PATH}.bak

YELLOW='\033[1;33m'
NC='\033[0m'

echo "${YELLOW}${SKYDB_REV_FILENAME}${NC} has been added to $SKYDB_MIGRATION_PATH"
echo "Please add ${YELLOW}&revision_${SKYDB_REV}{}${NC} to ${SKYDB_MIGRATION_PATH}/revision_list.go"
echo "Please replace revision of ${SKYDB_MIGRATION_PATH}/full.go ${YELLOW}{SKYDB_REV}${NC}"
