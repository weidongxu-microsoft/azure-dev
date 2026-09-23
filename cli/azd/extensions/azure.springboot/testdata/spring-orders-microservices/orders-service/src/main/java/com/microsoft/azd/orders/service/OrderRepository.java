// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.orders.service;

import java.math.BigDecimal;
import java.util.List;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

@Repository
public class OrderRepository {
    private final JdbcClient jdbc;

    public OrderRepository(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    public Order create(String sku, String itemName, int quantity, BigDecimal total) {
        return jdbc.sql("""
                INSERT INTO orders (sku, item_name, quantity, total)
                VALUES (:sku, :itemName, :quantity, :total)
                RETURNING id, sku, item_name, quantity, total
                """)
            .param("sku", sku)
            .param("itemName", itemName)
            .param("quantity", quantity)
            .param("total", total)
            .query(Order.class)
            .single();
    }

    public List<Order> list() {
        return jdbc.sql("SELECT id, sku, item_name, quantity, total FROM orders ORDER BY id")
            .query(Order.class)
            .list();
    }
}
