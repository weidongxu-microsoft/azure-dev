package com.microsoft.azd.springaiagent;

import java.util.function.Supplier;

import com.azure.core.credential.AccessToken;
import com.azure.core.credential.TokenCredential;
import com.azure.core.credential.TokenRequestContext;
import com.azure.identity.DefaultAzureCredentialBuilder;
import com.openai.client.OpenAIClient;
import com.openai.client.OpenAIClientAsync;
import com.openai.client.okhttp.OpenAIOkHttpClient;
import com.openai.client.okhttp.OpenAIOkHttpClientAsync;
import com.openai.credential.BearerTokenCredential;

import org.springframework.ai.chat.model.ChatModel;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.ai.openai.OpenAiChatOptions;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class FoundryModelConfiguration {

    private static final String FOUNDRY_SCOPE = "https://ai.azure.com/.default";

    @Bean
    TokenCredential tokenCredential() {
        return new DefaultAzureCredentialBuilder().build();
    }

    @Bean
    ChatModel chatModel(
        TokenCredential tokenCredential,
        @Value("${spring.ai.openai.base-url}") String endpoint,
        @Value("${spring.ai.openai.chat.options.model}") String model) {
        Supplier<String> tokenSupplier = () -> getToken(tokenCredential);
        OpenAIClient client = OpenAIOkHttpClient.builder()
            .baseUrl(endpoint)
            .credential(BearerTokenCredential.create(tokenSupplier))
            .build();
        OpenAIClientAsync asyncClient = OpenAIOkHttpClientAsync.builder()
            .baseUrl(endpoint)
            .credential(BearerTokenCredential.create(tokenSupplier))
            .build();

        return OpenAiChatModel.builder()
            .openAiClient(client)
            .openAiClientAsync(asyncClient)
            .options(OpenAiChatOptions.builder().model(model).build())
            .build();
    }

    private static String getToken(TokenCredential credential) {
        AccessToken token = credential.getToken(
            new TokenRequestContext().addScopes(FOUNDRY_SCOPE)).block();
        if (token == null) {
            throw new IllegalStateException("Microsoft Entra ID did not return a Foundry access token.");
        }
        return token.getToken();
    }
}
