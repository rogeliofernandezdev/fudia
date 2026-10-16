CREATE UNIQUE INDEX audit_log_concierge_bill_request_uq
ON audit_log(organization_id,location_id,(metadata->>'requestId'))
WHERE action='concierge.bill.requested' AND metadata ? 'requestId';
