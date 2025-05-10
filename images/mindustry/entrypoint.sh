#!/bin/ash
# shellcheck shell=dash

set -eu

MEMORY_OPTS=${MEMORY_OPTS:-"-Xms128M -Xmx1G"}
EXTRA_JAVA_OPTS=${EXTRA_JAVA_OPTS:-}
JAVA_OPTS="${MEMORY_OPTS} ${EXTRA_JAVA_OPTS}"

CONFIG_COMMANDS=""

# Iterate all MINDUSTRY_* environment variables.
for var_name in $(env | grep MINDUSTRY_ | cut -d= -f1); do
    # Strip the prefix
    raw_key=${var_name#MINDUSTRY_}

    # Convert to lowercase, then to camelCase
    camel_key=$(printf '%s\n' "$raw_key" | awk -F_ '{
        for (i=1; i<=NF; i++) {
            $i = tolower($i)
            if (i > 1) $i = toupper(substr($i,1,1)) substr($i,2)
        }
        printf "%s", $1
        for (i=2; i<=NF; i++) printf "%s", $i
    }')

    # Get the value of the variable
    var_value=$(eval "echo \$$var_name")

    CONFIG_COMMANDS="${CONFIG_COMMANDS}config ${camel_key} ${var_value}\n"
done

if [ -z "${CONFIG_COMMANDS}" ]; then
    echo "No MINDUSTRY_* environment variables found. Skipping configuration."
else
    # shellcheck disable=SC2086,SC3036
    echo -e "${CONFIG_COMMANDS}exit\n" | java ${JAVA_OPTS} -jar /mindustry-server.jar
fi

# shellcheck disable=SC2086
exec java ${JAVA_OPTS} -jar /mindustry-server.jar
