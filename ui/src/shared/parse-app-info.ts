function parseImageRepository(infoArray) {
  try {
    // Find the object with the name "Image Repository"
    const imageRepoEntry = infoArray.find((item) => item.name === "Image Repository");
    if (imageRepoEntry && imageRepoEntry.value) {
      // Extract the part before the colon
      return imageRepoEntry.value.split(":")[0] || null;
    }
  } catch (err) {
    console.error("Error parsing info array for Image Repository:", err);
  }
  return null; // Return null if parsing fails or value is not found
}


function parseAppRepository(infoArray) {
  try {
    // Find the object with the name "Application Repository"
    const appRepoEntry = infoArray.find((item) => item.name === "Application Repository");
    if (appRepoEntry && appRepoEntry.value) {
      // Return the value directly
      return appRepoEntry.value.trim() || null;
    }
  } catch (err) {
    console.error("Error parsing info array for Application Repository:", err);
  }
  return null; // Return null if parsing fails or value is not found
}


function findMatchingImage(images, repoName) {
  if (!repoName) return null;

  try {
    // Find the matching image
    const matchingImage = images.find((image) => image.startsWith(repoName));
    if (matchingImage) {
      // Extract the tag part between the colon and the "@" symbol
      const partsAfterColon = matchingImage.split(":");
      if (partsAfterColon.length > 1) {
        const tagPart = partsAfterColon[1].split("@")[0];
        return tagPart; // Return the part before the "@" symbol
      }
    }
  } catch (err) {
    console.error("Error processing images array:", err);
  }

  return null; // Return null if no match or parsing fails
}


export function getAppDetails(images, info) {
  // Extract the application repository
  const appRepository = parseAppRepository(info);

  console.info(appRepository)

  // Extract the image repository
  const imageRepository = parseImageRepository(info);

  // Find the matching image
  const imageTag = findMatchingImage(images, imageRepository);

  // Return both values as an object
  return {
    appRepository,
    imageTag,
  };
}