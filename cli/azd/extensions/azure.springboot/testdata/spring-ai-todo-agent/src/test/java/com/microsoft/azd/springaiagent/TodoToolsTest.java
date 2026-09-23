package com.microsoft.azd.springaiagent;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatIllegalArgumentException;

import org.junit.jupiter.api.Test;
import org.springframework.ai.support.ToolCallbacks;

class TodoToolsTest {

    private final TodoTools todoTools = new TodoTools();

    @Test
    void createsListsAndCompletesTodos() {
        TodoItem first = todoTools.createTodo("Write sample");
        TodoItem second = todoTools.createTodo("Run azd init");

        assertThat(todoTools.listTodos()).containsExactly(first, second);

        TodoItem completed = todoTools.completeTodo(first.id());

        assertThat(completed.completed()).isTrue();
        assertThat(todoTools.listTodos()).containsExactly(completed, second);
    }

    @Test
    void rejectsBlankTitles() {
        assertThatIllegalArgumentException()
            .isThrownBy(() -> todoTools.createTodo("  "))
            .withMessage("Todo title must not be blank.");
    }

    @Test
    void rejectsUnknownTodo() {
        assertThatIllegalArgumentException()
            .isThrownBy(() -> todoTools.completeTodo(42))
            .withMessage("Todo 42 does not exist.");
    }

    @Test
    void generatesToolSchemas() {
        assertThat(ToolCallbacks.from(todoTools)).hasSize(3);
    }
}
