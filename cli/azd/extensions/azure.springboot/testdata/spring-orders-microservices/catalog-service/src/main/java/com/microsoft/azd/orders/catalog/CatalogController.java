// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.orders.catalog;

import com.microsoft.azd.orders.contracts.CatalogItem;
import java.math.BigDecimal;
import java.util.Map;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/catalog")
public class CatalogController {
    private static final Map<String, CatalogItem> ITEMS = Map.of(
        "widget", new CatalogItem("widget", "Validation Widget", new BigDecimal("19.95")),
        "gadget", new CatalogItem("gadget", "Deployment Gadget", new BigDecimal("29.95")));

    @GetMapping("/{sku}")
    public ResponseEntity<CatalogItem> get(@PathVariable String sku) {
        var item = ITEMS.get(sku);
        return item == null ? ResponseEntity.notFound().build() : ResponseEntity.ok(item);
    }
}
