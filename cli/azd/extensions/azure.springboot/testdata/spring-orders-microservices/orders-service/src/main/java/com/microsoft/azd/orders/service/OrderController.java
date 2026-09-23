// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.orders.service;

import jakarta.validation.Valid;
import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/orders")
public class OrderController {
    private final CatalogClient catalog;
    private final OrderRepository orders;

    public OrderController(CatalogClient catalog, OrderRepository orders) {
        this.catalog = catalog;
        this.orders = orders;
    }

    @PostMapping
    public Order create(@Valid @RequestBody CreateOrderRequest request) {
        var item = catalog.getItem(request.sku());
        var total = item.price().multiply(java.math.BigDecimal.valueOf(request.quantity()));
        return orders.create(item.sku(), item.name(), request.quantity(), total);
    }

    @GetMapping
    public List<Order> list() {
        return orders.list();
    }
}
