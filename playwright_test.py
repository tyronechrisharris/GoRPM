from playwright.sync_api import sync_playwright
import time

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        page.goto("http://localhost:8080")

        # Wait a bit for JS to fetch and render
        time.sleep(2)

        page.screenshot(path="screenshot.png", full_page=True)

        browser.close()

if __name__ == "__main__":
    run()
