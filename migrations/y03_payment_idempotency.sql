USE `looklook_payment`;

-- Existing installations must resolve duplicate order/service rows before this
-- constraint can be added. Fresh installations already receive it from the
-- base schema, so this migration is safe to execute repeatedly.
SET @index_exists = (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = 'looklook_payment'
    AND table_name = 'third_payment'
    AND index_name = 'uk_order_service'
);
SET @statement = IF(
  @index_exists = 0,
  'ALTER TABLE `third_payment` ADD UNIQUE KEY `uk_order_service` (`order_sn`,`service_type`)',
  'SELECT 1'
);
PREPARE payment_idempotency_migration FROM @statement;
EXECUTE payment_idempotency_migration;
DEALLOCATE PREPARE payment_idempotency_migration;
