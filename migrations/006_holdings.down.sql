-- Откат нужен старому образу: на версии 6 он не стартует. Теряются все позиции и вся история снимков.

DROP TRIGGER update_holdings_updated_at;
DROP TABLE holding_values;
DROP TABLE holdings;
