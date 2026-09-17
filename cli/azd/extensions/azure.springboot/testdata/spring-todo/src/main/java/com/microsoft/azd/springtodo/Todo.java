// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.springtodo;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.Table;

@Table("todos")
public record Todo(
    @Id @Column("id") Long id,
    @Column("title") String title,
    @Column("completed") boolean completed) {
}
