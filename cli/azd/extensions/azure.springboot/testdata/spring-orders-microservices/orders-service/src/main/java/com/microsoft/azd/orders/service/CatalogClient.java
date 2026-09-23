// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.orders.service;

import com.microsoft.azd.orders.contracts.CatalogItem;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

@Component
public class CatalogClient {
    private final RestClient client;

    public CatalogClient(RestClient.Builder builder, @Value("${catalog.service.url}") String serviceUrl) {
        this.client = builder.baseUrl(serviceUrl).build();
    }

    public CatalogItem getItem(String sku) {
        return client.get()
            .uri("/api/catalog/{sku}", sku)
            .retrieve()
            .body(CatalogItem.class);
    }
}
