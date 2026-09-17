// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.springtodo;

import jakarta.validation.constraints.NotBlank;

public record CreateTodoRequest(@NotBlank String title) {
}
