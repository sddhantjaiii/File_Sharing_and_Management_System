import openai

# Replace with your actual API key
openai.api_key = "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"

def check_openai_key():
    try:
        response = openai.ChatCompletion.create(
            model="gpt-3.5-turbo",
            messages=[{"role": "user", "content": "Say 'Hello, OpenAI!'"}]
        )
        print("✅ API Key is valid!")
        print("Response from OpenAI:", response.choices[0].message.content)
    except openai.error.AuthenticationError:
        print("❌ Invalid API Key!")
    except Exception as e:
        print("⚠️ An error occurred:", e)

check_openai_key()
