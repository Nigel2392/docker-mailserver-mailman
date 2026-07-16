#!/bin/bash

echo "Applying custom Postfix quota hook..."
CURRENT_RESTRICTIONS=$(postconf -h smtpd_recipient_restrictions)

if [[ "$CURRENT_RESTRICTIONS" != *"check_policy_service inet:127.0.0.1:65265"* ]]; then
    postconf -e "smtpd_recipient_restrictions = ${CURRENT_RESTRICTIONS}, check_policy_service inet:127.0.0.1:65265"
    echo "Successfully appended quota policy to smtpd_recipient_restrictions."
fi

echo "Injecting Doveadm HTTP API Key..."

# this grabs the password from env and saves it natively into Dovecot
echo "doveadm_api_key = ${DOVEADM_PASSWORD}" > /etc/dovecot/conf.d/99-api.conf