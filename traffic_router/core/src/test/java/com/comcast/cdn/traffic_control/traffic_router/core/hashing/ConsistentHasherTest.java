/*
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.comcast.cdn.traffic_control.traffic_router.core.hashing;

import com.comcast.cdn.traffic_control.traffic_router.core.ds.Dispersion;
import com.comcast.cdn.traffic_control.traffic_router.core.hash.ConsistentHasher;
import com.comcast.cdn.traffic_control.traffic_router.core.hash.DefaultHashable;
import com.comcast.cdn.traffic_control.traffic_router.core.hash.Hashable;
import com.comcast.cdn.traffic_control.traffic_router.core.hash.MD5HashFunction;
import com.comcast.cdn.traffic_control.traffic_router.core.hash.NumberSearcher;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.Before;
import org.junit.Test;
import org.mockito.InjectMocks;
import org.mockito.Mock;

import java.util.List;
import java.util.ArrayList;
import java.util.Map;
import java.util.HashMap;
import java.util.Random;

import static org.hamcrest.Matchers.greaterThan;
import static org.hamcrest.Matchers.lessThan;
import static org.hamcrest.core.AnyOf.anyOf;
import static org.hamcrest.core.IsEqual.equalTo;
import static org.junit.Assert.assertThat;
import static org.mockito.MockitoAnnotations.initMocks;

public class ConsistentHasherTest {
	@Mock
	MD5HashFunction md5HashFunction = new MD5HashFunction();

	@Mock
	NumberSearcher numberSearcher = new NumberSearcher();

	@InjectMocks
	DefaultHashable hashable1 = new DefaultHashable();

	@InjectMocks
	DefaultHashable hashable2 = new DefaultHashable();

	@InjectMocks
	DefaultHashable hashable3 =  new DefaultHashable();

	List<DefaultHashable> hashables = new ArrayList<DefaultHashable>();

	@InjectMocks
	ConsistentHasher consistentHasher;

	@Before
	public void before() {
		hashable1.generateHashes("hashId1", 100);
		hashable2.generateHashes("hashId2", 100);
		hashable3.generateHashes("hashId3", 100);

		hashables.add(hashable1);
		hashables.add(hashable2);
		hashables.add(hashable3);

		initMocks(this);
	}

	@Test
	public void itHashes() throws Exception {
		final ObjectMapper mapper = new ObjectMapper();
		DefaultHashable hashable = consistentHasher.selectHashable(hashables, new Dispersion(mapper.createObjectNode()), "some-string");
		assertThat(hashable, anyOf(equalTo(hashable1), equalTo(hashable2), equalTo(hashable3)));
		DefaultHashable nextHashable = consistentHasher.selectHashable(hashables, new Dispersion(mapper.createObjectNode()),"some-string");
		assertThat(nextHashable, equalTo(hashable));
	}

	@Test
	public void itHashesMoreThanOne() throws Exception {
		final String jsonStr = "{\"dispersion\": {\n" +
				"\"limit\": 2,\n" +
				"\"shuffled\": \"true\"\n" +
				"}}";
		final ObjectMapper mapper = new ObjectMapper();
		final JsonNode jo = mapper.readTree(jsonStr);
		Dispersion dispersion = new Dispersion(jo);

		List<DefaultHashable> results = consistentHasher.selectHashables(hashables, dispersion, "some-string");
		assertThat(results.size(), equalTo(2));
		assertThat(results.get(0), anyOf(equalTo(hashable1), equalTo(hashable2), equalTo(hashable3)));
		assertThat(results.get(1), anyOf(equalTo(hashable1), equalTo(hashable2), equalTo(hashable3)));
		List<DefaultHashable> results2 = consistentHasher.selectHashables(hashables, dispersion, "some-string");
		assert(results.containsAll(results2));

		final String jsonStr2 = "{\"dispersion\": {\n" +
				"\"limit\": 2000000000,\n" +
				"\"shuffled\": \"true\"\n" +
				"}}";
		final JsonNode jo2 = mapper.readTree(jsonStr2);
		Dispersion disp2 = new Dispersion(jo2);
		List <DefaultHashable> res3 = consistentHasher.selectHashables(hashables, disp2, "some-string");
		assert(res3.containsAll(hashables));

	}


	@Test
	public void itemsMigrateFromSmallerToLargerBucket() {
		List<String> randomPaths = new ArrayList<>();

		for (int i = 0; i < 10000; i++) {
			randomPaths.add(generateRandomPath());
		}

		Hashable smallerBucket = new DefaultHashable().generateHashes("Small One", 10000);
		Hashable largerBucket = new DefaultHashable().generateHashes("Larger bucket", 90000);

		List<Hashable> buckets = new ArrayList<>();
		buckets.add(smallerBucket);
		buckets.add(largerBucket);

		Map<Hashable, List<String>> hashedPaths = new HashMap<>();
		hashedPaths.put(smallerBucket, new ArrayList<String>());
		hashedPaths.put(largerBucket, new ArrayList<String>());

		final ObjectMapper mapper = new ObjectMapper();
		for (String randomPath : randomPaths) {
			Hashable hashable = consistentHasher.selectHashable(buckets, new Dispersion(mapper.createObjectNode()), randomPath);
			hashedPaths.get(hashable).add(randomPath);
		}

		Hashable grownBucket = new DefaultHashable().generateHashes("Small One", 20000);
		Hashable shrunkBucket = new DefaultHashable().generateHashes("Larger bucket", 80000);

		List<Hashable> changedBuckets = new ArrayList<>();
		changedBuckets.add(grownBucket);
		changedBuckets.add(shrunkBucket);

		Map<Hashable, List<String>> rehashedPaths = new HashMap<>();
		rehashedPaths.put(grownBucket, new ArrayList<String>());
		rehashedPaths.put(shrunkBucket, new ArrayList<String>());

		for (String randomPath : randomPaths) {
			Hashable hashable = consistentHasher.selectHashable(changedBuckets, new Dispersion(mapper.createObjectNode()), randomPath);
			rehashedPaths.get(hashable).add(randomPath);
		}

		assertThat(rehashedPaths.get(grownBucket).size(), greaterThan(hashedPaths.get(smallerBucket).size()));
		assertThat(rehashedPaths.get(shrunkBucket).size(), lessThan(hashedPaths.get(largerBucket).size()));

		for (String path : hashedPaths.get(smallerBucket)) {
			assertThat(rehashedPaths.get(grownBucket).contains(path), equalTo(true));
		}

		for (String path : rehashedPaths.get(shrunkBucket)) {
			assertThat(hashedPaths.get(largerBucket).contains(path), equalTo(true));
		}
	}

	String alphanumericCharacters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWZYZ";
	String exampleValidPathCharacters = alphanumericCharacters + "/=;()-.";

	Random random = new Random(1462307930227L);
	String generateRandomPath() {
		int pathLength = 60 + random.nextInt(61);

		StringBuilder stringBuilder = new StringBuilder("/");
		for (int i = 0; i < 4; i++) {
			int index = random.nextInt(alphanumericCharacters.length());
			stringBuilder.append(alphanumericCharacters.charAt(index));
		}

		stringBuilder.append("/");

		for (int i = 0; i < pathLength; i++) {
			int index = random.nextInt(exampleValidPathCharacters.length());
			stringBuilder.append(exampleValidPathCharacters.charAt(index));
		}

		return stringBuilder.toString();
	}
}
