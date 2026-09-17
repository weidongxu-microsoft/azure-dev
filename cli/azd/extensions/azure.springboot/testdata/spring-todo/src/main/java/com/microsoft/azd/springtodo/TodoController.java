// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package com.microsoft.azd.springtodo;

import java.util.Comparator;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.atomic.AtomicLong;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.server.ResponseStatusException;

import jakarta.validation.Valid;

@RestController
@RequestMapping("/api/todos")
public class TodoController {

    private final AtomicLong nextId = new AtomicLong();
    private final ConcurrentMap<Long, Todo> todos = new ConcurrentHashMap<>();

    @GetMapping
    public List<Todo> list() {
        return todos.values().stream()
            .sorted(Comparator.comparingLong(Todo::id))
            .toList();
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public Todo create(@Valid @RequestBody CreateTodoRequest request) {
        long id = nextId.incrementAndGet();
        Todo todo = new Todo(id, request.title(), false);
        todos.put(id, todo);
        return todo;
    }

    @PutMapping("/{id}/complete")
    public Todo complete(@PathVariable long id) {
        return todos.compute(id, (ignored, todo) -> {
            if (todo == null) {
                throw new ResponseStatusException(HttpStatus.NOT_FOUND, "Todo not found");
            }

            return new Todo(todo.id(), todo.title(), true);
        });
    }
}
