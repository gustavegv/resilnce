# Problems after stage 1 (resolved)
Stage 1 has been implemented, and I have tested it. The implementation faces several severe problems.

The entire point of the Stage 1 implementation was a non-invasive implementation of the new DB structure (new structure in 'docs/exercise-centric/schema.md'), but as some features were required for the new schema slight changes and additions would have to be made.

Once testing however, I have noticed that the changes are quite significant in certain places, which has completely changed how the application looks.

## Problems
1. The workout screen is completely changed. The previous version used a component which was the tracking screen (see image 'docs/exercise-centric/images/resilnce-old.png'). The "new" version is completely stripped out of style, and does not re-use that component at all (see image 'docs/exercise-centric/images/resilnce-new.png'), why is this when the implementation was supposed to be as un-invasive as possible? If additions are required, add them to were they were in the previous design. Don't rewrite a page layout. The same goes for the finish screen, not being re-used at all. This saddens me as I have put a lot of blood, sweat and tears into the design being as it is. It might only be that the styles somehow are not applied, but I dont see how that would be the case if we are using components.

2. There are multiple other places where the style has been completely stripped, i.e the continue workout button.

3. There are multiple other places where the function has been completely stripped, i.e the exercise buttons in /tracker page not having the three dots for slide-opening the delete or edit menu button underneath. Here the edit screen has also been completely destroyed, instead of being re-used.

## Context
Please look at the main branch, for context on how the frontend should look and function, and RE-USE the Svelte components which are used in the main branch! If additional features need to be built on top of these components, add them to the component without disturbing its current layout. It is mostly the Exercise