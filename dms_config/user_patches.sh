#!/bin/bash
# user-patches.sh

echo "Applying custom Postfix quota hook..."

# Query Postfix for the actively generated restrictions
CURRENT_RESTRICTIONS=$(postconf -h smtpd_recipient_restrictions)

# Append the quota hook, ensuring we don't duplicate it if the container restarts
if [[ "$CURRENT_RESTRICTIONS" != *"check_policy_service inet:127.0.0.1:65265"* ]]; then
    postconf -e "smtpd_recipient_restrictions = ${CURRENT_RESTRICTIONS}, check_policy_service inet:127.0.0.1:65265"
    echo "Successfully appended quota policy to smtpd_recipient_restrictions."
else
    echo "Quota policy already present. Skipping."
fi