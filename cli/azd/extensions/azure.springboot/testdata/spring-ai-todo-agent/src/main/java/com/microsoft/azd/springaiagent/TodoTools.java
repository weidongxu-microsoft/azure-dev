package com.microsoft.azd.springaiagent;

import java.util.Comparator;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.atomic.AtomicLong;

import org.springframework.ai.tool.annotation.Tool;
import org.springframework.ai.tool.annotation.ToolParam;
import org.springframework.stereotype.Component;

@Component
public class TodoTools {

    private final AtomicLong nextId = new AtomicLong();
    private final ConcurrentMap<Long, TodoItem> todos = new ConcurrentHashMap<>();

    @Tool(description = "List all todo items and their completion state.")
    public List<TodoItem> listTodos() {
        return todos.values().stream()
            .sorted(Comparator.comparingLong(TodoItem::id))
            .toList();
    }

    @Tool(description = "Create a new todo item.")
    public TodoItem createTodo(@ToolParam(description = "The todo title.") String title) {
        String normalizedTitle = title == null ? "" : title.trim();
        if (normalizedTitle.isEmpty()) {
            throw new IllegalArgumentException("Todo title must not be blank.");
        }

        long id = nextId.incrementAndGet();
        TodoItem todo = new TodoItem(id, normalizedTitle, false);
        todos.put(id, todo);
        return todo;
    }

    @Tool(description = "Mark an existing todo item as completed.")
    public TodoItem completeTodo(@ToolParam(description = "The todo identifier.") long id) {
        return todos.compute(id, (ignored, todo) -> {
            if (todo == null) {
                throw new IllegalArgumentException("Todo " + id + " does not exist.");
            }
            return new TodoItem(todo.id(), todo.title(), true);
        });
    }
}
