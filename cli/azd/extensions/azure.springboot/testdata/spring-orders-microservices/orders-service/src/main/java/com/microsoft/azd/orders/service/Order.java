// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.orders.service;

import java.math.BigDecimal;

public record Order(long id, String sku, String itemName, int quantity, BigDecimal total) {
}
