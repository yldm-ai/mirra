# Changelog

## Unreleased

- A toast's close button is named for assistive technology, which read it as an unnamed button.

## 0.20.0 - 2026-11-20

- 130 backgrounds instead of 14, most of them outdoors: beaches, lakes, mountains, forests, fields, deserts, streets, old towns and harbors, besides a few rooms.
- The website and the web app moved to https://mirra.yldm.ai and https://mirra.yldm.ai/app/; the old addresses lead there.
- The background's library tab reads Scene in English and German, which fit it on one line beside Face, Outfit and Pose.

## 0.19.0 - 2026-11-20

- The settings are laid out as cards of rows, each setting's name on the left and its control on the right, on the desktop and the phone alike. The AI service page opens on the chosen provider with its logo, the connection's state and the test, and a failed test says why across the card; the advanced settings and Reset to defaults follow the same cards.
- General adds a switch for the galleries' 18+ categories: on asks your age once, off forgets the answer, so turning it back on asks again. Before, the answer could not be taken back.
- About opens on the app's icon and name, and Sources lists where the backgrounds come from.

## 0.18.0 - 2026-11-20

### Features

- Set the look in a background

## 0.17.0 - 2026-11-20

- The app speaks Traditional Chinese, Japanese, Korean, Spanish, French, German, Brazilian Portuguese, Russian and Italian as well as English and Simplified Chinese, chosen in the language setting or following the system.
- A fourth element, the background: indoor, city and nature scenes the app ships, public domain photographs with no one in them. None is chosen by default and the style sets the scene; a chosen one is sent with the look and the person is set there. A work keeps its background, and pressing the chosen one again or its × goes back to none.

## 0.16.0 - 2026-11-20

- The model is chosen from a searchable list of the image models the key can use, fetched from the provider once the key is entered or the connection test passes, with a button to fetch it again, on the same page as the provider and the key; it was typed under Advanced. Alibaba Cloud Model Studio and Azure OpenAI, whose models a key can't list, still take it typed.

## 0.15.0 - 2026-11-20

- EveryAPI is preset among the AI services, with its logo and a link to get a key; it starts on `gpt-image-2` through its OpenAI-compatible API, and its China address can be set under Advanced.

## 0.14.0 - 2026-11-20

- A style for the picture beside Generate: studio, street, white backdrop, film or anime. It is kept for the next launch and with each work, which opens again in its style.

## 0.13.0 - 2026-11-20

- Settings have a language: follow the system, English or 简体中文, under General on the desktop and in Me on the phone. The app switches at once, keeping what is open, and keeps the choice for the next launch.

## 0.12.0 - 2026-11-20

- When a service's safety check refuses a picture, the error says so and suggests another outfit, pose or service, with the service's reason below; it read as a refused request. OpenAI, Azure OpenAI, Volcengine Ark, Alibaba Cloud Model Studio and Gemini refusals are recognized.
- The desktop studio's three columns are resized by dragging the gutters between them. The library and the look open at a share of the window, wider on a larger screen, and keep the width they were dragged to as a share of the window; the library shows more columns of pictures as it widens.

## 0.11.0 - 2026-11-20

- Surprise me, the dice beside the studio's title on the phone and the canvas's zoom on the desktop, chooses a face, an outfit and a pose at random from what each gallery shows, adults-only items only where they are turned on.
- The desktop settings put their parts on tabs down the side: AI service, Advanced, About and Sources, where they were one long column with the advanced settings folded away.

## 0.10.0 - 2026-11-20

- All angles, beside the pose's angle chips, makes one sheet of the look from every camera angle of the pose side by side, the same person and outfit in each, on a landscape canvas. The angles go to the image service joined into one reference, which any service takes; a work keeps its angles, and Generate again makes the sheet again. Two-person poses make one picture as before.

## 0.9.1 - 2026-11-20

- On macOS, the app menu has About, Settings… (⌘,) and Check for Updates…, with the Services, Hide and Window items Mac apps have; it held only Quit.
## 0.9.0 - 2026-11-20

- The update prompts show what the new version changes: its section of this changelog, in Sparkle's and WinSparkle's dialogs on macOS and Windows and below the version on Android.

## 0.8.2 - 2026-11-20

- The answer to the 18+ switch's age question is kept, as works and favorites from those categories show on later launches; it was asked again every launch.
- Errors say what went wrong in the app's language, such as a refused key, a model that isn't found, no credit left, a timeout or no network, with what the service said below, where they showed its raw message or Go's network error in English.
- Azure OpenAI asks for the deployment's name beside its key and address, where it was hidden under Advanced, and OpenRouter's connection test checks the key, which listing its models never did.

## 0.8.1 - 2026-11-20

- Two-person poses ask the image model for two adults, with the chosen face and outfit on the main one, where the prompt asked for one person.

## 0.8.0 - 2026-11-20

- The app updates itself from the latest release: Sparkle on macOS and WinSparkle on Windows check daily, Android as it starts, and About has Check for updates. Releases carry the signed appcasts the installed copies read; copies of 0.7.0 and older are reinstalled once.

## 0.7.0 - 2026-11-20

- The AI settings show the provider and key first, with the address, model and size under Advanced and a reset to the provider's defaults; the size is chosen as portrait, square or landscape, the key is shown with an eye button, the About section gives the version, and the desktop settings scroll when they are taller than the window.

## 0.6.0 - 2026-11-20

- The AI settings preset OpenAI, Google Gemini, Volcengine Ark, xAI, OpenRouter, Alibaba Cloud Model Studio, Azure OpenAI and stable-diffusion.cpp, with their logos and a link to get a key, besides any OpenAI-compatible address. Each provider keeps its own key and settings; the cloud or local service set up before carries over.

## 0.5.0 - 2026-11-20

- Lingerie and sexy outfits, and revealing and two-person poses, show only behind each gallery's 18+ switch, which first asks whether you are 18 or older; the lingerie and sexy chips are translated.

## 0.4.0 - 2026-11-19

- Releases on GitHub with a signed and notarized macOS disk image, a signed Windows installer and the web build, made from `main` when its commits call for a version; the website's download cards link them.
- The app is named Mirra, capitalized, on the web page and in the installed apps.
- A web version at https://yldm-ai.github.io/mirra/app/, published with the website; its key, works and clothes last as long as the page.
- A website at https://yldm-ai.github.io/mirra/ in Chinese, English and Japanese, with share cards, language alternates and a sitemap for search engines.
- The Chinese README's phone screenshot, which was blank, shows the studio again.

## 0.3.0

- A dark amber look with serif titles, after the second reference design.
- The AI service is a cloud API or a local server (whose key may be left out), with a connection test and its status at the top of the app; on the desktop it is set up in a dialog.
- A new picture waits to be saved to Works; generate it again, swap its outfit with one tap, star, export or delete it.
- Works are numbered, can be starred and filtered, and open again with the look they were made from.
- The wardrobe keeps favorite outfits and photos of the user's own clothes, and shows what the look is wearing.
- The desktop canvas zooms, and changes the face or pose from its corner.
- The picture count is gone from the interface; one Generate makes one picture.

## 0.2.0

- Renamed Mirra (镜衣), with a warm theme in light and dark.
- A phone layout (Studio, Wardrobe, Works, Me; the look put together in three steps) and a desktop studio of three columns, chosen by the window's width.
- Faces filter by hair length and come from the face gallery's front-facing section only; poses filter by camera angle.
- Ask for 1, 2 or 4 pictures at once, and cancel a request while it runs.
- Photo mode is gone: the app takes no photo of the user, as the pose and outfit galleries' terms forbid depicting real people with their images.

## 0.1.0

- Combine a pose, an outfit and a face from three reference galleries into one picture through an OpenAI-compatible `/images/edits` endpoint.
- Give your own photo a face from the face gallery; outfits go on poses only, as the outfit gallery's terms require.
- Works kept on the device, with save, share and delete; the API key in the system keychain.
- English and Simplified Chinese.
- macOS, iOS, Android and Windows builds.
