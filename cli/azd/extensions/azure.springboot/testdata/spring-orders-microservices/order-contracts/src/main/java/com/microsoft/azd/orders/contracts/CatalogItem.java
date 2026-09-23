// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.orders.contracts;

import java.math.BigDecimal;

public record CatalogItem(String sku, String name, BigDecimal price) {
}
